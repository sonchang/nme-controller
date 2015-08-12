package nme

type Lbvserver struct {
	Name string
	IpAddress string
	Port string
	Bindings map[string]Service
}

type Service struct {
	Name string
	IpAddress string
}

type LbConfigs struct {
	Hash string
	NSIPs []string
	LbMaps map[string]Lbvserver
}
