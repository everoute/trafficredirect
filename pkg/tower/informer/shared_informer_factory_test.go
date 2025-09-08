package informer

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/everoute/graphc/pkg/informer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

func TestInformer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "tower informer Suite")
}

var _ = Describe("sharedInformerFactory", func() {
	var (
		f *sharedInformerFactory
	)

	BeforeEach(func() {
		f = &sharedInformerFactory{
			vnicCrcChan: make(chan *informer.CrcEvent, 10),
		}
	})

	Describe("convertObj", func() {
		It("should return nil if obj is nil", func() {
			Expect(f.convertObj(nil)).To(BeNil())
		})

		It("should return VMNic struct if json is valid", func() {
			v := &datamodel.VmNic{ObjectMeta: datamodel.ObjectMeta{ID: "vnic-1"}}
			b, _ := json.Marshal(v)
			s := string(b)
			fmt.Printf("--- %v", s)

			res := f.convertObj(&s)
			Expect(res).NotTo(BeNil())
			Expect(res.ID).To(Equal("vnic-1"))
		})

		It("should return nil if json invalid", func() {
			s := "{invalid-json}"
			res := f.convertObj(&s)
			Expect(res).To(BeNil())
		})
	})

	Describe("handleCrc", func() {
		var (
			resourceType string
			resourceID   string
		)

		BeforeEach(func() {
			rt := string(datamodel.TypeVMNic)
			resourceType = rt
			resourceID = "id-1"
		})

		It("should skip nil event", func() {
			f.handleCrc(nil)
			Expect(f.vnicCrcChan).To(BeEmpty())
		})

		It("should skip event with wrong resource type", func() {
			action := string(informer.CrcEventInsert)
			rt := "VM"
			e := &models.ResourceChangeEvent{
				Action:       &action,
				ResourceType: &rt,
				ResourceID:   &resourceID,
			}
			f.handleCrc(e)
			Expect(f.vnicCrcChan).To(BeEmpty())
		})

		It("should handle insert event", func() {
			action := string(informer.CrcEventInsert)
			newVnic := &datamodel.VmNic{ObjectMeta: datamodel.ObjectMeta{ID: "new"}}
			b, _ := json.Marshal(newVnic)
			newStr := string(b)

			e := &models.ResourceChangeEvent{
				Action:       &action,
				ResourceType: &resourceType,
				ResourceID:   &resourceID,
				NewValue:     &newStr,
			}

			f.handleCrc(e)
			Expect(f.vnicCrcChan).NotTo(BeEmpty())
			ev := <-f.vnicCrcChan
			Expect(ev.EventType).To(Equal(informer.CrcEventInsert))
			Expect(ev.NewObj.(*datamodel.VmNic).ID).To(Equal("new"))
		})

		It("should handle update event", func() {
			action := string(informer.CrcEventUpdate)
			newVnic := &datamodel.VmNic{ObjectMeta: datamodel.ObjectMeta{ID: "new"}}
			oldVnic := &datamodel.VmNic{ObjectMeta: datamodel.ObjectMeta{ID: "old"}}
			newStr, _ := json.Marshal(newVnic)
			oldStr, _ := json.Marshal(oldVnic)
			newVal := string(newStr)
			oldVal := string(oldStr)

			e := &models.ResourceChangeEvent{
				Action:       &action,
				ResourceType: &resourceType,
				ResourceID:   &resourceID,
				NewValue:     &newVal,
				OldValue:     &oldVal,
			}

			f.handleCrc(e)
			Expect(f.vnicCrcChan).NotTo(BeEmpty())
			ev := <-f.vnicCrcChan
			Expect(ev.EventType).To(Equal(informer.CrcEventUpdate))
			Expect(ev.NewObj.(*datamodel.VmNic).ID).To(Equal("new"))
			Expect(ev.OldObj.(*datamodel.VmNic).ID).To(Equal("old"))
		})

		It("should handle delete event", func() {
			action := string(informer.CrcEventDelete)
			oldVnic := &datamodel.VmNic{ObjectMeta: datamodel.ObjectMeta{ID: "old"}}
			oldStr, _ := json.Marshal(oldVnic)
			oldVal := string(oldStr)

			e := &models.ResourceChangeEvent{
				Action:       &action,
				ResourceType: &resourceType,
				ResourceID:   &resourceID,
				OldValue:     &oldVal,
			}

			f.handleCrc(e)
			Expect(f.vnicCrcChan).NotTo(BeEmpty())
			ev := <-f.vnicCrcChan
			Expect(ev.EventType).To(Equal(informer.CrcEventDelete))
			Expect(ev.OldObj.(*datamodel.VmNic).ID).To(Equal("old"))
		})

		It("should skip unknown action", func() {
			action := "UNKNOWN"
			e := &models.ResourceChangeEvent{
				Action:       &action,
				ResourceType: &resourceType,
				ResourceID:   &resourceID,
			}

			f.handleCrc(e)
			Expect(f.vnicCrcChan).To(BeEmpty())
		})
	})
})
