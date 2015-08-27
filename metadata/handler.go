package metadata

import (
	"encoding/json"
	"fmt"
	"strings"
	"io/ioutil"
	"net/http"
	log "github.com/sonchang/nme-controller/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/sonchang/nme-controller/nme"
)

type MetadataHandler struct {
	metadataUrl string
}

type ExtLbConfig struct {
	stackName string
	serviceName string
	publicPort string
	privatePort string
	serviceType string
}

func NewHandler(url string) MetadataHandler {
	return MetadataHandler{
		metadataUrl: url,
	}
}

func (m *MetadataHandler) GetHash() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/version", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Debugf("resp body=%s", string(body[:]))
	return string(body[:]), nil
}

// Make HTTP GET from metadata, parse JSON results, and map to LB mappings
func (m *MetadataHandler) GetLbConfig() (map[string]nme.Lbvserver, error) {
	client := &http.Client{}
	serviceReq, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/services", nil)
	if err != nil {
		return nil, err
	}
	serviceReq.Header.Add("Accept", "application/json")
	serviceResp, err := client.Do(serviceReq)
	defer serviceResp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(serviceResp.Body)
	if err != nil {
		return nil, err
	}
	log.Debugf("resp body=%s", string(body[:]))
	var serviceMappings interface{}
	err = json.Unmarshal(body, &serviceMappings)
	if err != nil {
		return nil, err
	}

	services, ok := serviceMappings.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unexpected JSON results from metadata service: %v", serviceMappings)
	}

	containerReq, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/containers", nil)
	if err != nil {
		return nil, err
	}
	containerReq.Header.Add("Accept", "application/json")
	containerResp, err := client.Do(containerReq)
	defer containerResp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err = ioutil.ReadAll(containerResp.Body)
	if err != nil {
		return nil, err
	}
	log.Debugf("resp body=%s", string(body[:]))
	var containerMappings interface{}
	err = json.Unmarshal(body, &containerMappings)
	if err != nil {
		return nil, err
	}

	containers, ok := containerMappings.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unexpected JSON results from metadata service: %v", containerMappings)
	}

	lbConfig, err := m.getLbConfigFromJSONResults(services, containers)
	return lbConfig, err
}

func (m *MetadataHandler) getLbConfigFromJSONResults(services, containers []interface{}) (map[string]nme.Lbvserver, error) {
	newConfig := make(map[string]nme.Lbvserver)
	nmeServices := make(map[string]nme.Service)

	for i := range containers {
		container, ok := containers[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing container data: %v", containers[i])
		}
		name := container["name"].(string)
		// for native docker containers, looks like primary_ip can be nil
		ip, ok := container["primary_ip"].(string)
		if ok {
			serviceBinding := nme.Service {
				Name: name,
				IpAddress: ip,
			}
			nmeServices[name] = serviceBinding
		}
	}

	serviceToPortConfig, err := m.getServiceToPortConfigsFromVipService(services)
	if err != nil {
		return nil, err
	}
	for i := range services {
		service, ok := services[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing service data: %v", services[i])
		}
		labels, ok := service["labels"].(map[string]interface{})
		if ok {
			netscalerService, ok := labels["io.rancher.netscaler.me"].(string)
			if ok {
				log.Debugf("skipping netscaler service: %v", service)
				continue
			}
		}
		lbvserver, err := m.getLbvserverFromService(service, nmeServices, serviceToPortConfig)
		if err != nil {
			return nil, err
		}
		newConfig[(*lbvserver).Name] = *lbvserver
	}


	return newConfig, nil
}

func (m *MetadataHandler) getServiceToPortConfigsFromVipService(services []interface{}) (map[string]ExtLbConfig, error) {
	for i := range services {
		service, ok := services[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing service data: %v", services[i])
		}
		labels, ok := service["labels"].(map[string]interface{})
		if !ok {
			log.Debugf("skipping %v", service)
			continue
		}
		networkServices, ok := labels["io.rancher.network.services"]
		if !ok || !strings.Contains(networkServices.(string), "vipService") {
			log.Debugf("skipping %v", service)
			continue
		}
		extLbConfigs, ok := labels["io.rancher.netscaler.lb"].(string)
		if ok {
			log.Debugf("found netscaler external lb configs: %v", extLbConfigs)
			serviceToPortConfigs := make(map[string]ExtLbConfig)
			configs := strings.Split(extLbConfigs, ",")
			for i := range configs {
				log.Debugf("configs %v", configs)
				extLbConfigPieces := strings.Split(configs[i], ":")
				switch len(extLbConfigPieces) {
				case 2, 3:
					stackServiceName := strings.Split(extLbConfigPieces[0], "/")
					if len(stackServiceName) != 2 {
						continue
					}
					var portProtocol []string
					var publicPort string
					if len(extLbConfigPieces) == 2 {
						portProtocol = strings.Split(extLbConfigPieces[1], "/")
					} else if len(extLbConfigPieces) == 3 {
						publicPort = extLbConfigPieces[1]
						portProtocol = strings.Split(extLbConfigPieces[2], "/")
					}
					privatePort := portProtocol[0]
					serviceType := "ANY"
					if len(portProtocol) > 1 {
						serviceType = strings.ToUpper(portProtocol[1])
					}
					if publicPort == "" {
						publicPort = privatePort
					}
					extLbConfig := ExtLbConfig {
						stackName: stackServiceName[0],
						serviceName: stackServiceName[1],
						publicPort: publicPort,
						privatePort: privatePort,
						serviceType: serviceType,
					}
					log.Debugf("extLbConfig: %v", extLbConfig)
					serviceToPortConfigs[extLbConfig.stackName + "/" + extLbConfig.serviceName] = extLbConfig
				}
			}
			return serviceToPortConfigs, nil
		}
		return nil, nil
	}
	return nil, nil
}

func (m *MetadataHandler) getLbvserverFromService(
		service map[string]interface{},
		nmeServices map[string]nme.Service,
		serviceToPortConfig map[string]ExtLbConfig) (*nme.Lbvserver, error) {

	stackName := service["stack_name"].(string)
	serviceName := service["name"].(string)
	stackServiceName := stackName + "/" + serviceName

	vip := service["vip"].(string)
	containers := service["containers"].([]interface{})
	serviceBindings := make(map[string]nme.Service)

	var publicPort string
	var privatePort string
	var serviceType string
	extLbConfig, ok := serviceToPortConfig[stackServiceName]
	if ok {
		publicPort = extLbConfig.publicPort
		privatePort = extLbConfig.privatePort
		serviceType = extLbConfig.serviceType
	}

	for i := range containers {
		containerName := containers[i].(string)
		service, ok := nmeServices[containerName]
		if ok {
			if privatePort != "" {
				service.Port = privatePort
			} else if publicPort != "" {
				service.Port = publicPort
			} else {
				service.Port = "*"
			}
			serviceBindings[containerName] = service
		}
	}

	lbvserver := nme.Lbvserver {
		Name: stackName + "_" + serviceName, 
		IpAddress: vip,
		Bindings: serviceBindings,
	}

	if serviceType == "" {
		lbvserver.ServiceType = "ANY"
	} else {
		lbvserver.ServiceType = serviceType
	}

	if publicPort == "" {
		lbvserver.Port = "*"
	} else {
		lbvserver.Port = publicPort
	}
	return &lbvserver, nil
}
