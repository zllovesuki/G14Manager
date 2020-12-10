package announcement

type UpdateType int

const (
	FeaturesUpdate UpdateType = iota
	ProfilesUpdate
)

type Update struct {
	Type   UpdateType
	Config interface{}
}

type Updatable interface {
	Name() string
	ConfigUpdate(c Update)
}
