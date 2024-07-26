package network

type Service uint64
type Services = Service

const (
	None           Service = 0
	Network        Service = 1
	GetUTXO        Service = 2
	Bloom          Service = 4
	Witness        Service = 8
	Xthin          Service = 16
	CompactFilters Service = 32
	NetworkLimited Service = 64
)
