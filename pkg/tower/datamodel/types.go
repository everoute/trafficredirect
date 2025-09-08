package datamodel

type ResourceType string

const TypeVMNic ResourceType = "VmNic"

type VmNic struct {
	ObjectMeta

	DPIEnabled bool   `json:"dpi_enabled,omitempty"`
	MacAddress string `json:"mac_address,omitempty"`
	VM         VM     `json:"vm,omitempty"`
}

type VM struct {
	ID string `json:"id"`
}
