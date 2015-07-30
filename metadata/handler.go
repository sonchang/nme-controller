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

func (m MetadataHandler) GetHash() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/hash", nil)
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
func (m MetadataHandler) GetLbConfig() (map[string]nme.Lbvserver, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + m.metadataUrl + "/latest/stacks", nil)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Debugf("resp body=%s", string(body[:]))
	var mappings interface{}
	err = json.Unmarshal(body, &mappings)
	if err != nil {
		return nil, err
	}

	// this will likely change
	stacks, ok := mappings.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unexpected JSON results from metadata service: %v", mappings)
	}
	lbConfig, err := m.getLbConfigFromStacks(stacks)
	return lbConfig, err
}

func (m MetadataHandler) getLbConfigFromStacks(stacks []interface{}) (map[string]nme.Lbvserver, error) {
	newConfig := make(map[string]nme.Lbvserver)

	for i := range stacks {
		stack := stacks[i]
		stackMetadata, ok := stack.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing stack data: %v", stack)
		}
		services, ok := stackMetadata["services"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing services data: %v", stackMetadata["services"])
		}
		if services == nil {
			continue
		}

		for j := range services {
			service, ok := services[j].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Error parsing service data: %v", services[j])
			}
			lbvserver, err := m.getLbvserverFromService(service)
			if err != nil {
				return nil, err
			}
			newConfig[(*lbvserver).Name] = *lbvserver
		}
	}
	return newConfig, nil
}

func (m MetadataHandler) getLbvserverFromService(service map[string]interface{}) (*nme.Lbvserver, error) {
	// TODO: if serviceName does not include stack name, prepend it
	serviceName := service["name"].(string)
	vip := service["ip"].(string)
	serviceBindings, err := m.getServiceBindingsFromContainers(service["containers"].([]interface{}))
	if err != nil {
		return nil, err
	}

	lbvserver := nme.Lbvserver {
		Name: serviceName, 
		IpAddress: vip,
		 Bindings: serviceBindings,
	}
	return &lbvserver, nil
}

func (m MetadataHandler) getServiceBindingsFromContainers(containers []interface{}) (map[string]nme.Service, error) {
	bindings := make(map[string]nme.Service)
	for i := range containers {
		container, ok := containers[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Error parsing container data: %v", containers[i])
		}
		name := container["name"].(string)
		ip := container["ip"].(string)
		serviceBinding := nme.Service {
			Name: name,
			 IpAddress: ip,
		}
		bindings[name] = serviceBinding
	}
	return bindings, nil
}
