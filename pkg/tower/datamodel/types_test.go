package datamodel

import (
	"reflect"
	"testing"

	graphcinformer "github.com/everoute/graphc/pkg/informer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	VMNicGqlType = "vmNic"
)

func TestType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "tower type Suite")
}

var _ = Describe("ResourceType String", func() {
	It("should return correct gql type", func() {
		gt := graphcinformer.NewGqlType(reflect.TypeOf(&VmNic{}))
		Expect(gt.TypeName()).Should(Equal(VMNicGqlType))
		Expect(gt.QueryFields(nil)).Should(Equal("{id,dpi_enabled,mac_address,vm{id}}"))
	})
})
