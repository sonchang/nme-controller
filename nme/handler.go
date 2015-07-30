package nme

type NmeHandler struct {
	apiHandler NitroApi
}

func NewHandler(nitroApi NitroApi) NmeHandler {
	return NmeHandler{
		apiHandler: nitroApi,
	}
}

func (n NmeHandler) AddNSIP(ip string) error {
	return n.apiHandler.AddNSIP(ip)
}

func (n NmeHandler) CreateLB(lb Lbvserver) error {
	err := n.apiHandler.CreateLbvserver(lb.Name, lb.IpAddress)
	if err != nil {
		return err
	}
	for _, service := range lb.Bindings {
		err := n.CreateServiceAndBinding(lb, service)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n NmeHandler) DeleteLB(lb Lbvserver) error {
	return n.apiHandler.DeleteLbvserver(lb.Name)
}

func (n NmeHandler) CreateServiceAndBinding(lb Lbvserver, service Service) error {
	err := n.apiHandler.CreateService(service.Name, service.IpAddress)
	if err != nil {
		return err
	}
	err = n.apiHandler.BindServiceToLbvserver(lb.Name, service.Name)
	if err != nil {
		return err
	}
	return nil
}

// not sure if Nitro supports updating the IP.  I know it supports updating
// resources, but when I tried updating the "ip" field, it rejected it
func (n NmeHandler) UpdateServiceIp(lb Lbvserver, service Service) error {
	err := n.DeleteServiceAndBinding(service)
	if err != nil {
		return err
	}
	return n.CreateServiceAndBinding(lb, service)
}

func (n NmeHandler) DeleteServiceAndBinding(service Service) error {
	return n.apiHandler.DeleteService(service.Name)
}
