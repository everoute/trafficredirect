package datamodel

type ResourceType string

const (
	TypeVMNic ResourceType = "VmNic"
)

type GqlType interface {
	GqlGetStr(id string) string
	TypeName() string
}
