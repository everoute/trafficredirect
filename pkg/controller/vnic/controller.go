package vnic

import (
	"context"
	"fmt"
	"time"

	graphcinformer "github.com/everoute/graphc/pkg/informer"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/constants"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
	"github.com/everoute/trafficredirect/pkg/tower/informer"
)

const (
	CrcChanSize = 100
)

type Controller struct {
	k8scli k8sclient.Client
	ruleW  controller.Controller

	vnicCache     graphcinformer.Lister
	vnicHasSynced func() bool

	queue workqueue.RateLimitingInterface
}

func NewController(mgr ctrl.Manager, fac informer.SharedInformerFactory) *Controller {
	c := &Controller{
		k8scli: mgr.GetClient(),
		queue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	vnicInformer := fac.VmNic()
	vnicInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.vnicAdd,
		UpdateFunc: c.vnicUpdate,
		DeleteFunc: c.vnicDelete,
	})
	c.vnicCache = vnicInformer.GetIndexer()
	c.vnicHasSynced = vnicInformer.HasSynced

	var err error
	c.ruleW, err = controller.NewUnmanaged("rule", mgr, controller.Options{Reconciler: reconcile.Func(c.handleRule)})
	if err != nil {
		klog.Fatalf("Failed to new rule controller: %s", err)
	}
	err = c.ruleW.Watch(source.Kind(mgr.GetCache(), &v1alpha1.Rule{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		klog.Fatalf("Failed to watch rule: %s", err)
	}

	return c
}

func (c *Controller) Start(stopCtx context.Context) error {
	defer c.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("vnic-rule", stopCtx.Done(), c.vnicHasSynced) {
		return fmt.Errorf("failed to wait cache sync for vnic-rule controller")
	}

	g, ctx := errgroup.WithContext(stopCtx)

	g.Go(func() error {
		return c.ruleW.Start(ctx)
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

func (c *Controller) handle(ctx context.Context, vnicID string) error {
	klog.V(2).Infof("Begin to process vnic %s", vnicID)
	defer klog.V(2).Infof("End to process vnic %s", vnicID)

	obj, exists, err := c.vnicCache.GetByKey(vnicID)
	if err != nil {
		klog.Errorf("Failed to get vnic %s from cache: %s", vnicID, err)
		return err
	}
	if !exists {
		klog.Infof("Vnic %s doesn't exists, try to delete related rule", vnicID)
		ruleI := VnicIDToRuleName(vnicID, v1alpha1.Ingress)
		if err := c.deleteRule(ctx, ruleI); err != nil {
			return err
		}
		ruleE := VnicIDToRuleName(vnicID, v1alpha1.Egress)
		if err := c.deleteRule(ctx, ruleE); err != nil {
			return err
		}
		return nil
	}
	klog.Infof("Vnic %s exists, try to add or update related rule", vnicID)
	vnic, ok := obj.(*datamodel.VmNic)
	if !ok || vnic == nil {
		klog.Errorf("Invalid vnic object %v", obj)
		return fmt.Errorf("invalid vnic object")
	}
	if vnic.DPIEnabled {
		klog.Infof("Vnic %s enable DPI, try to add or update related rule", vnicID)
		ruleI := VnicToRule(vnic, v1alpha1.Ingress)
		if err := c.addOrUpdateRule(ctx, ruleI); err != nil {
			return err
		}
		ruleE := VnicToRule(vnic, v1alpha1.Egress)
		if err := c.addOrUpdateRule(ctx, ruleE); err != nil {
			return err
		}
		return nil
	}

	klog.Infof("Vnic %s disable DPI, try to delete related rule", vnicID)
	ruleI := VnicIDToRuleName(vnicID, v1alpha1.Ingress)
	if err := c.deleteRule(ctx, ruleI); err != nil {
		return err
	}
	ruleE := VnicIDToRuleName(vnicID, v1alpha1.Egress)
	if err := c.deleteRule(ctx, ruleE); err != nil {
		return err
	}
	return nil
}

func (c *Controller) handleRule(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ruleID := RuleNameToVnicID(req.Name)
	if ruleID == "" || req.Namespace != constants.VnicRuleNamespace {
		klog.Infof("Rule %s is not from tower vm nic, skip", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	c.queue.Add(ruleID)
	klog.Infof("Success to add vnic %s to queue from rule %s", ruleID, req.NamespacedName)
	return ctrl.Result{}, nil
}

func (c *Controller) deleteRule(ctx context.Context, n string) error {
	k := types.NamespacedName{Namespace: constants.VnicRuleNamespace, Name: n}
	rule := &v1alpha1.Rule{}
	if err := c.k8scli.Get(ctx, k, rule); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		klog.Errorf("Failed to get rule %s: %s", k, err)
		return err
	}
	if err := c.k8scli.Delete(ctx, rule); err != nil {
		klog.Errorf("Failed to delete rule %v: %s", *rule, err)
		return err
	}
	klog.Infof("Success to delete rule %v", *rule)
	return nil
}

func (c *Controller) addOrUpdateRule(ctx context.Context, nRule *v1alpha1.Rule) error {
	k := types.NamespacedName{Namespace: nRule.GetNamespace(), Name: nRule.GetName()}
	rule := &v1alpha1.Rule{}
	if err := c.k8scli.Get(ctx, k, rule); err != nil {
		if errors.IsNotFound(err) {
			if err := c.k8scli.Create(ctx, nRule); err != nil {
				klog.Errorf("Failed to create rule %v: %s", nRule, err)
				return err
			}
			klog.Infof("Success to create rule %v", nRule)
			return nil
		}
		klog.Errorf("Failed to get rule %s: %s", k, err)
		return err
	}

	if equality.Semantic.DeepEqual(rule.Spec, nRule.Spec) {
		return nil
	}
	rule.Spec = nRule.Spec
	if err := c.k8scli.Update(ctx, rule); err != nil {
		klog.Errorf("Failed to update rule %v: %s", rule, err)
		return err
	}
	klog.Infof("Success to update rule %v", rule)
	return nil
}

func (c *Controller) vnicAdd(obj any) {
	vnic, ok := obj.(*datamodel.VmNic)
	if !ok || vnic == nil {
		klog.Errorf("Failed to convert to vmnic object for add, skip, newObj: %v", obj)
		return
	}
	c.queue.Add(vnic.GetID())
}

func (c *Controller) vnicUpdate(oldObj, newObj any) {
	newVnic, ok := newObj.(*datamodel.VmNic)
	if !ok || newVnic == nil {
		klog.Errorf("Failed to convert to vmnic object for update, skip, newObj: %v", newObj)
		return
	}
	oldVnic, ok := oldObj.(*datamodel.VmNic)
	if !ok || oldVnic == nil {
		klog.Errorf("Failed to convert to vmnic object for update, skip, oldObj: %v", oldObj)
		return
	}
	if *newVnic == *oldVnic {
		return
	}
	c.queue.Add(newVnic.GetID())
}

func (c *Controller) vnicDelete(obj any) {
	oldVnic, ok := obj.(*datamodel.VmNic)
	if !ok || oldVnic == nil {
		klog.Errorf("Failed to convert to vmnic object for delete, skip, oldObj: %v", obj)
		return
	}
	c.queue.Add(oldVnic.GetID())
}
