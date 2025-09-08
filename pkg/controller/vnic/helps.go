package vnic

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/constants"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

func vnicIDToRuleName(vnicID string, d v1alpha1.RuleDirect) string {
	return fmt.Sprintf("%s-%s-%s", constants.VnicRulePrefix, vnicID, d)
}

func ruleNameToVnicID(n string) string {
	parts := strings.Split(n, "-")
	if len(parts) != 3 || parts[0] != constants.VnicRulePrefix {
		return ""
	}

	if parts[2] != string(v1alpha1.Ingress) && parts[2] != string(v1alpha1.Egress) {
		return ""
	}

	return parts[1]
}

func vnicToRule(vnic *datamodel.VMNic, d v1alpha1.RuleDirect) *v1alpha1.Rule {
	name := vnicIDToRuleName(vnic.GetID(), d)
	rule := &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: constants.VnicRuleNamespace},
		Spec: v1alpha1.RuleSpec{
			Direct: d,
			Option: &v1alpha1.Option{
				TowerVM: vnic.VM.ID,
			},
		},
	}
	if d == v1alpha1.Egress {
		rule.Spec.Match.SrcMac = vnic.MacAddress
	}
	if d == v1alpha1.Ingress {
		rule.Spec.Match.DstMac = vnic.MacAddress
	}
	return rule
}
