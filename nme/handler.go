package nme

import (
	log "github.com/Sirupsen/logrus"
)

type NmeHandler struct {
	apiHandler NitroApi
}

const (
	defaultDockerSNIP = "172.17.0.200"
)

func NewHandler(nitroApi NitroApi) NmeHandler {
	return NmeHandler{
		apiHandler: nitroApi,
	}
}

func (n NmeHandler) LoadConfigs() (map[string]Lbvserver, error) {
	lbvservers, err := n.apiHandler.GetLbvservers()
	if err != nil {
		return nil, err
	}
	lbConfigs := make(map[string]Lbvserver)
	for lbvserverName, lbvserverMap := range lbvservers {
		vip := lbvserverMap["ipaddress"]
		port := lbvserverMap["port"]

		lbvServerBindings := make(map[string]Service)
		bindings, err := n.apiHandler.GetLbvserverBindings(lbvserverName)
		if err != nil {
			log.Errorf("Error getting service bindings for %s: %v", lbvserverName, err)
			continue;
		}
		for serviceName, serviceDetailsMap := range bindings {
			serviceBinding := Service{
				Name: serviceName,
				IpAddress: serviceDetailsMap["ipaddress"],
			}
			lbvServerBindings[serviceName] = serviceBinding
		}

		lbConfig := Lbvserver{
			Name: lbvserverName,
			IpAddress: vip,
			Port: port,
			Bindings: lbvServerBindings,
		}
		lbConfigs[lbvserverName] = lbConfig
	}
	return lbConfigs, nil
}

func (n NmeHandler) UpdateNSIP(ipAddresses ...string) error {
	currentSNIPMap, err := n.apiHandler.GetSNIPs()
	if err != nil {
		return err
	}

	newSNIPMap := make(map[string]bool)
	for i := range ipAddresses {
		newSNIPMap[ipAddresses[i]] = true
		//if _, ok := currentSNIPMap[ipAddresses[i]]; !ok {
		if currentSNIPMap[ipAddresses[i]] == false {
			err := n.apiHandler.AddNSIP(ipAddresses[i])
			if err != nil {
				return err
			}
		}
	}
	for snip := range currentSNIPMap {
		//if _, ok := newSNIPMap[snip]; !ok {
		if newSNIPMap[snip] == false && snip != defaultDockerSNIP {
			err := n.apiHandler.DeleteNSIP(snip)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
