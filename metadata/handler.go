package metadata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	log "github.com/Sirupsen/logrus"

	"github.com/sonchang/nme-controller/nme"
)

type MetadataHandler struct {
	metadataUrl string
}

func NewHandler(url string) MetadataHandler {
	return MetadataHandler{
		metadataUrl: url,
	}
}

func (m *MetadataHandler) GetHash() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/updated", nil)
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
		stackServiceName := container["name"].(string) //container["stackName"].(string) + "/" + container["serviceName"].(string)
		name := container["name"].(string)
		ip := container["primaryIp"].(string)
		serviceBinding := nme.Service {
			Name: name,
			IpAddress: ip,
		}
		nmeServices[stackServiceName] = serviceBinding
	}

	for i := range services {
		service, ok := services[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing service data: %v", services[i])
		}
		lbvserver, err := m.getLbvserverFromService(service, nmeServices)
		if err != nil {
			return nil, err
		}
		newConfig[(*lbvserver).Name] = *lbvserver
	}


	return newConfig, nil
}

func (m *MetadataHandler) getLbvserverFromService(service map[string]interface{}, nmeServices map[string]nme.Service) (*nme.Lbvserver, error) {
	serviceName := service["name"].(string)
	vip := service["vip"].(string)
	containers := service["containers"].([]interface{})
	serviceBindings := make(map[string]nme.Service)

	for i := range containers {
		containerName := containers[i].(string)
		service := nmeServices[containerName]
		serviceBindings[containerName] = service
	}

	lbvserver := nme.Lbvserver {
		Name: serviceName, 
		IpAddress: vip,
		Bindings: serviceBindings,
	}
	return &lbvserver, nil
}

