package datamodel

// Object lets you work with object metadata from tower.
type Object interface {
	// GetID returns the object ID.
	GetID() string
}

// ObjectMeta is metadata that all tower resources must have.
type ObjectMeta struct {
	// ID is the unique in time and space value for this object
	ID string `json:"id"`
}

// GetID returns the object ID.
func (obj *ObjectMeta) GetID() string { return obj.ID }

// ObjectReference is the reference to other object
type ObjectReference ObjectMeta
