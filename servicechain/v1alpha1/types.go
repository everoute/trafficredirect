package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=trafficredirectrules,shortName=trr
// +kubebuilder:printcolumn:name="src-mac",type="string",JSONPath=".spec.match.srcMac"
// +kubebuilder:printcolumn:name="dst-mac",type="string",JSONPath=".spec.match.dstMac"
// +kubebuilder:printcolumn:name="vm",type="string",JSONPath=".spec.towerOption.vmID"
// +kubebuilder:printcolumn:name="vnic",type="string",JSONPath=".spec.towerOption.nic"

type TrafficRedirectRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior for this TrafficRedirect.
	Spec TrafficRedirectRuleSpec `json:"spec"`
}

type TrafficRedirectRuleSpec struct {
	Match   TrafficRedirectRuleMatch `json:"match"`
	Egress  bool                     `json:"egress,omitempty"`
	Ingress bool                     `json:"ingress,omitempty"`
	// tower info for debug
	TowerOption *TowerOption `json:"towerOption,omitempty"`
}

type TrafficRedirectRuleMatch struct {
	SrcMac string `json:"srcMac,omitempty"`
	DstMac string `json:"dstMac,omitempty"`
}

type TowerOption struct {
	VMID string `json:"vmID,omitempty"`
	Nic  string `json:"nic,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TrafficRedirectRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrafficRedirectRule `json:"items"`
}
