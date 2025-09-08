package datamodel

import (
	"fmt"
)

const (
	VMNicGqlTypeName = "vmNic"
	VMNicGqlFields   = "{id,dpi_enabled,mac_address,vm{id}}"
)

type VMNic struct {
	ObjectMeta

	DPIEnabled bool   `json:"dpi_enabled,omitempty"`
	MacAddress string `json:"mac_address,omitempty"`
	VM         VM     `json:"vm,omitempty"`
}

type VM struct {
	ID string `json:"id"`
}

func (r VMNic) GqlGetStr(id string) string {
	return fmt.Sprintf("query {%s(where:{id:\"%s\"}) %s}", VMNicGqlTypeName, id, VMNicGqlFields)
}

func (r VMNic) TypeName() string {
	return VMNicGqlTypeName
}
