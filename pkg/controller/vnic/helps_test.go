package vnic

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1alpha1 "github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/constants"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

func TestVnic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vnic Suite")
}

var _ = Describe("Vnic Helper Functions", func() {

	Describe("VnicIDToRuleName", func() {
		It("should generate correct rule name", func() {
			ruleName := VnicIDToRuleName("vnic-1", v1alpha1.Ingress)
			Expect(ruleName).To(Equal(constants.VnicRulePrefix + "-vnic-1-ingress"))

			ruleName = VnicIDToRuleName("vnic-2", v1alpha1.Egress)
			Expect(ruleName).To(Equal(constants.VnicRulePrefix + "-vnic-2-egress"))
		})
	})

	Describe("RuleNameToVnicID", func() {
		It("should return vnicID for valid rule name", func() {
			vnicID := RuleNameToVnicID(constants.VnicRulePrefix + "-vnic123-ingress")
			Expect(vnicID).To(Equal("vnic123"))

			vnicID = RuleNameToVnicID(constants.VnicRulePrefix + "-vnic456-egress")
			Expect(vnicID).To(Equal("vnic456"))
		})

		It("should return empty string for invalid rule name", func() {
			Expect(RuleNameToVnicID("invalid-name")).To(BeEmpty())
			Expect(RuleNameToVnicID(constants.VnicRulePrefix + "-vnic-789-unknown")).To(BeEmpty())
		})
	})

	Describe("VnicToRule", func() {
		var testVnic *datamodel.VmNic

		BeforeEach(func() {
			testVnic = &datamodel.VmNic{
				ObjectMeta: datamodel.ObjectMeta{
					ID: "vnic-1",
				},
				MacAddress: "aa:bb:cc:dd:ee:ff",
				VM: datamodel.VM{
					ID: "vm-123",
				},
			}
		})

		It("should generate ingress rule correctly", func() {
			rule := VnicToRule(testVnic, v1alpha1.Ingress)
			Expect(rule.Name).To(Equal(constants.VnicRulePrefix + "-vnic-1-ingress"))
			Expect(rule.Namespace).To(Equal(constants.VnicRuleNamespace))
			Expect(rule.Spec.Direct).To(Equal(v1alpha1.Ingress))
			Expect(rule.Spec.Match.DstMac).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(rule.Spec.Match.SrcMac).To(BeEmpty())
			Expect(rule.Spec.Option.TowerVM).To(Equal("vm-123"))
		})

		It("should generate egress rule correctly", func() {
			rule := VnicToRule(testVnic, v1alpha1.Egress)
			Expect(rule.Name).To(Equal(constants.VnicRulePrefix + "-vnic-1-egress"))
			Expect(rule.Namespace).To(Equal(constants.VnicRuleNamespace))
			Expect(rule.Spec.Direct).To(Equal(v1alpha1.Egress))
			Expect(rule.Spec.Match.SrcMac).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(rule.Spec.Match.DstMac).To(BeEmpty())
			Expect(rule.Spec.Option.TowerVM).To(Equal("vm-123"))
		})
	})
})
