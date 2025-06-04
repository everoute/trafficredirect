package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rules,shortName=trr
// +kubebuilder:printcolumn:name="src-mac",type="string",JSONPath=".spec.match.srcMac"
// +kubebuilder:printcolumn:name="dst-mac",type="string",JSONPath=".spec.match.dstMac"
// +kubebuilder:printcolumn:name="vm",type="string",JSONPath=".spec.towerOption.vmID"
// +kubebuilder:printcolumn:name="vnic",type="string",JSONPath=".spec.towerOption.nic"

type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior for this Rule.
	Spec RuleSpec `json:"spec"`
}

type RuleSpec struct {
	Match RuleMatch `json:"match"`
	// +kubebuilder:validation:Enum=ingress;egress
	Direct RuleDirect `json:"direct"`
	// tower info for debug
	TowerOption *TowerOption `json:"towerOption,omitempty"`
}

type RuleMatch struct {
	SrcMac string `json:"srcMac,omitempty"`
	DstMac string `json:"dstMac,omitempty"`
}

type TowerOption struct {
	VMID string `json:"vmID,omitempty"`
	Nic  string `json:"nic,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rule `json:"items"`
}

type RuleDirect string

const (
	Egress  RuleDirect = "egress"
	Ingress RuleDirect = "ingress"
)
