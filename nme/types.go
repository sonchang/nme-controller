package nme

type Lbvserver struct {
	Name string
	IpAddress string
	Bindings map[string]Service
}

type Service struct {
	Name string
	IpAddress string
}

type LbConfigs struct {
	Hash string
	LbConfig map[string]Lbvserver
}
