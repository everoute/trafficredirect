package vnic

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/everoute/graphc/pkg/crcwatch"
	graphcinformer "github.com/everoute/graphc/pkg/informer"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/constants"
	ilog "github.com/everoute/trafficredirect/pkg/log"
	"github.com/everoute/trafficredirect/pkg/source"
	"github.com/everoute/trafficredirect/pkg/tower/client"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

const (
	CrcChanSize = 100
)

type Controller struct {
	towerCli  *client.Client
	k8scli    k8sclient.Client
	ruleW     controller.Controller
	crcW      *crcwatch.Watch
	syncCache cache.Cache

	queue workqueue.RateLimitingInterface
}

func NewController(mgr ctrl.Manager, towerCli *client.Client) *Controller {
	c := &Controller{
		towerCli: towerCli,
		k8scli:   mgr.GetClient(),
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	var err error
	c.ruleW, err = controller.NewUnmanaged("rule", mgr, controller.Options{Reconciler: reconcile.Func(c.ruleHandle)})
	if err != nil {
		ctrl.Log.Error(err, "Failed to new rule controller")
		os.Exit(1)
	}
	err = c.ruleW.Watch(source.Kind(mgr.GetCache(), &v1alpha1.Rule{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		ctrl.Log.Error(err, "Failed to watch rule")
		os.Exit(1)
	}
	c.syncCache = mgr.GetCache()

	c.crcW, err = client.NewCRCWatch([]datamodel.ResourceType{datamodel.TypeVMNic})
	if err != nil {
		ctrl.Log.Error(err, "Failed to new crc watch")
		os.Exit(1)
	}
	c.crcW.RegistryHandler(c.crcHandler)

	return c
}

func (c *Controller) Start(stopCtx context.Context) error {
	defer c.queue.ShutDown()

	if !c.syncCache.WaitForCacheSync(stopCtx) {
		return fmt.Errorf("timeout waiting for cache sync")
	}

	g, ctx := errgroup.WithContext(stopCtx)

	g.Go(func() error {
		return c.ruleW.Start(ctx)
	})

	g.Go(func() error {
		c.crcW.Start(ctx.Done())
		return nil
	})

	g.Go(func() error {
		wait.Until(
			graphcinformer.ReconcileWorker(ctx, "rule-sync", c.queue, c.handle),
			time.Second,
			ctx.Done(),
		)
		return nil
	})

	return g.Wait()
}

func (c *Controller) crcHandler(e *models.ResourceChangeEvent) {
	log := ctrl.Log.WithName("crcwatch")
	if e == nil {
		log.Info("crc event is nil, skip")
		return
	}
	log = log.WithValues("revision", *e.Revision)

	if e.Action == nil || e.ResourceType == nil || e.ResourceID == nil {
		log.Info("Invalid crc event, skip", "event", *e)
		return
	}

	if *e.ResourceType != string(datamodel.TypeVMNic) {
		log.Info("Unexpected resource type for crc event, skip", "event type", *e.ResourceType)
		return
	}

	log.V(4).Info("Received crc event", "type", *e.ResourceType, "id", *e.ResourceID, "action", *e.Action)
	c.queue.Add(*e.ResourceID)
}

func (c *Controller) ruleHandle(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("Reconciling rule start")
	defer log.V(4).Info("Reconciling rule end")

	vnicID := ruleNameToVnicID(req.Name)
	if vnicID == "" || req.Namespace != constants.VnicRuleNamespace {
		log.Info("Rule is not from tower vm nic, skip")
		return ctrl.Result{}, nil
	}
	c.queue.Add(vnicID)
	log.V(2).Info("Success to add vnic to queue from rule", "vnicID", vnicID)
	return ctrl.Result{}, nil
}

func (c *Controller) handle(ctx context.Context, vnicID string) error {
	ctx, log := ilog.GetAndSetLogForCtx(ctx, "handlerID", uuid.NewUUID(), "vnicID", vnicID)
	log.V(4).Info("Handling vnic start")
	defer log.V(4).Info("Handling vnic end")

	vnic := &datamodel.VMNic{}
	exists, err := c.towerCli.Get(ctx, vnicID, vnic)
	if err != nil {
		log.Error(err, "Failed to get vnic from tower")
		return err
	}
	if !exists {
		ctx, log := ilog.GetAndSetLogForCtx(ctx, "syncReason", "vnic not exists")
		log.V(4).Info("Vnic not exists, try to delete related rule")
		ruleI := vnicIDToRuleName(vnicID, v1alpha1.Ingress)
		if err := c.deleteRule(ctx, ruleI); err != nil {
			return err
		}
		ruleE := vnicIDToRuleName(vnicID, v1alpha1.Egress)
		if err := c.deleteRule(ctx, ruleE); err != nil {
			return err
		}
		return nil
	}

	if vnic.DPIEnabled {
		ctx, log := ilog.GetAndSetLogForCtx(ctx, "syncReason", "vnic dpi enabled")
		log.V(4).Info("Vnic DPI enabled, try to add or update related rule")
		ruleI := vnicToRule(vnic, v1alpha1.Ingress)
		if err := c.addOrUpdateRule(ctx, ruleI); err != nil {
			return err
		}
		ruleE := vnicToRule(vnic, v1alpha1.Egress)
		if err := c.addOrUpdateRule(ctx, ruleE); err != nil {
			return err
		}
		return nil
	}

	ctx, log = ilog.GetAndSetLogForCtx(ctx, "syncReason", "vnic dpi disabled")
	log.V(4).Info("Vnic DPI disabled, try to delete related rule")
	ruleI := vnicIDToRuleName(vnicID, v1alpha1.Ingress)
	if err := c.deleteRule(ctx, ruleI); err != nil {
		return err
	}
	ruleE := vnicIDToRuleName(vnicID, v1alpha1.Egress)
	if err := c.deleteRule(ctx, ruleE); err != nil {
		return err
	}
	return nil
}

func (c *Controller) deleteRule(ctx context.Context, n string) error {
	k := types.NamespacedName{Namespace: constants.VnicRuleNamespace, Name: n}
	log := ctrl.LoggerFrom(ctx, "ruleKey", k)
	rule := &v1alpha1.Rule{}
	if err := c.k8scli.Get(ctx, k, rule); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		log.Error(err, "Failed to get rule")
		return err
	}
	if err := c.k8scli.Delete(ctx, rule); err != nil {
		log.Error(err, "Failed to delete rule", "rule", *rule)
		return err
	}
	log.Info("Success to delete rule", "rule", *rule)
	return nil
}

func (c *Controller) addOrUpdateRule(ctx context.Context, nRule *v1alpha1.Rule) error {
	k := types.NamespacedName{Namespace: nRule.GetNamespace(), Name: nRule.GetName()}
	log := ctrl.LoggerFrom(ctx, "ruleKey", k)
	rule := &v1alpha1.Rule{}
	if err := c.k8scli.Get(ctx, k, rule); err != nil {
		if errors.IsNotFound(err) {
			if err := c.k8scli.Create(ctx, nRule); err != nil {
				log.Error(err, "Failed to create rule", "rule", *nRule)
				return err
			}
			log.Info("Success to create rule", "rule", *nRule)
			return nil
		}
		log.Error(err, "Failed to get rule")
		return err
	}

	if equality.Semantic.DeepEqual(rule.Spec, nRule.Spec) {
		return nil
	}
	rule.Spec = nRule.Spec
	if err := c.k8scli.Update(ctx, rule); err != nil {
		log.Error(err, "Failed to update rule", "rule", rule.Spec)
		return err
	}
	log.Info("Success to update rule", "rule", rule.Spec)
	return nil
}
