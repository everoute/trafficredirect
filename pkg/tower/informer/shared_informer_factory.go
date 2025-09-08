package informer

import (
	"context"
	"encoding/json"

	"github.com/everoute/graphc/pkg/crcwatch"
	graphcinformer "github.com/everoute/graphc/pkg/informer"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/everoute/trafficredirect/pkg/config"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

const (
	CrcChanSize = 100
)

type SharedInformerFactory interface {
	Start(context.Context) error
	VmNic() cache.SharedIndexInformer
}

type sharedInformerFactory struct {
	fac         graphcinformer.SharedInformerFactory
	crcW        *crcwatch.Watch
	vnicCrcChan chan *graphcinformer.CrcEvent
}

func NewSharedInformerFactory() SharedInformerFactory {
	cli := NewTowerClient()
	crcW, err := NewCRCWatch([]datamodel.ResourceType{datamodel.TypeVMNic})
	if err != nil || crcW == nil {
		klog.Fatalf("Failed to init crc watch, err: %s", err)
	}
	f := &sharedInformerFactory{
		fac:         *graphcinformer.NewSharedInformerFactory(cli, config.Config.Tower.ResyncPeriod),
		crcW:        crcW,
		vnicCrcChan: make(chan *graphcinformer.CrcEvent, CrcChanSize),
	}
	f.crcW.RegistryHandler(f.handleCrc)
	return f
}

func (f *sharedInformerFactory) Start(ctx context.Context) error {
	defer close(f.vnicCrcChan)

	f.fac.Start(ctx.Done())
	f.crcW.Start(ctx.Done())
	<-ctx.Done()
	return nil
}

func (f *sharedInformerFactory) VmNic() cache.SharedIndexInformer {
	return f.fac.InformerForWithCrc(&datamodel.VmNic{}, f.vnicCrcChan)
}

func (f *sharedInformerFactory) handleCrc(e *models.ResourceChangeEvent) {
	if e == nil {
		klog.Error("crc event is nil, skip")
		return
	}

	if e.Action == nil || e.ResourceType == nil || e.ResourceID == nil {
		klog.Errorf("Invalid crc event %v, skip", *e)
		return
	}

	klog.V(8).Infof("Received crc event: type: %s, id: %s, action: %v", *e.ResourceType, *e.ResourceID, *e.Action)
	if *e.ResourceType != string(datamodel.TypeVMNic) {
		klog.Errorf("Unexpected resource type %s for crc event, skip", *e.ResourceType)
		return
	}

	switch graphcinformer.CrcEventType(*e.Action) {
	case graphcinformer.CrcEventInsert:
		newObj := f.convertObj(e.NewValue)
		if newObj == nil {
			return
		}
		crce := graphcinformer.CrcEvent{EventType: graphcinformer.CrcEventInsert, NewObj: newObj}
		f.vnicCrcChan <- &crce
		return
	case graphcinformer.CrcEventUpdate:
		newObj := f.convertObj(e.NewValue)
		OldObj := f.convertObj(e.OldValue)
		if newObj == nil || OldObj == nil {
			return
		}
		crce := graphcinformer.CrcEvent{EventType: graphcinformer.CrcEventUpdate, NewObj: newObj, OldObj: OldObj}
		f.vnicCrcChan <- &crce
		return
	case graphcinformer.CrcEventDelete:
		if e.OldValue == nil {
			klog.Errorf("Nil old value for delete crc event, skip")
			return
		}
		oldObj := f.convertObj(e.OldValue)
		if oldObj == nil {
			return
		}
		crce := graphcinformer.CrcEvent{EventType: graphcinformer.CrcEventDelete, OldObj: oldObj}
		f.vnicCrcChan <- &crce
	default:
		klog.Errorf("Unknown crc action %s for crc event, skip", *e.Action)
		return
	}
}

func (f *sharedInformerFactory) convertObj(obj *string) *datamodel.ObjectMeta {
	if obj == nil {
		klog.Error("Nil obj to convert, skip")
		return nil
	}
	m := &datamodel.ObjectMeta{}
	if err := json.Unmarshal([]byte(*obj), m); err != nil {
		klog.V(3).Infof("Failed to unmarshal vnic, skip, obj %v, err: %s", *obj, err)
		return nil
	}
	klog.V(8).Infof("Convert crc value from %s to %v", obj, *m)
	return m
}
