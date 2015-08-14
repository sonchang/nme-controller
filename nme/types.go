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

type NmeApi interface {
	GetLbvservers() (map[string]map[string]string, error)
	GetLbvserverBindings(lbvserverName string) (map[string]map[string]string, error)
	GetSNIPs() (map[string]bool, error)
	DeleteNSIP(ip string) error
	AddNSIP(ip string) error
	CreateService(name string, ip string) error
	DeleteService(name string) error
	CreateLbvserver(lbvserverName string, vip string) error
	DeleteLbvserver(name string) error
	BindServiceToLbvserver(lbServiceName string, individualServiceName string) error
}