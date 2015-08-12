package nme

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"time"
	log "github.com/Sirupsen/logrus"
)

type NitroApi struct {
	nitroBaseUrl string
}

func NewNitroApi(baseUrl string) NitroApi {
	return NitroApi{
		nitroBaseUrl: baseUrl,
	}
}

func (n NitroApi) executeRequest(method string, url string, contentType string, jsonContent string) (interface{}, error) {
	log.Debugf("HTTP %s to %s, contents=%s", method, url, jsonContent)
	client := &http.Client{}
	req, err := http.NewRequest(method, "http://" + n.nitroBaseUrl + url, strings.NewReader(jsonContent))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-NITRO-USER", "root")
	req.Header.Add("X-NITRO-PASS", "linux")
	req.Header.Add("Content-Type", contentType)
	var attempts float64 = 1
	for {
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		log.Debugf("response StatusCode: %v", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debugf("response Body: %v", string(body))
		if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 409 {
			return body, nil
		}
		millis := math.Min(60000, math.Pow(2, attempts) * 1000)
		log.Debugf("waiting %v millis", millis)
		time.Sleep(time.Duration(millis) * time.Millisecond)
		attempts++
	}

	return nil, err
}

func (n NitroApi) GetLbvservers() (map[string]map[string]string, error) {
	resp, err := n.executeRequest("GET", "/nitro/v1/config/lbvserver", "", "")
	if err != nil {
		return nil, err
	}
	var jsonMap interface{}
	err = json.Unmarshal(resp.([]byte), &jsonMap)
	if err != nil {
		return nil, err
	}

	lbvservers := make(map[string]map[string]string)
	lbvserverMaps, ok := jsonMap.(map[string]interface{})["lbvserver"].([]interface{})
	if !ok {
		return lbvservers, nil
	}
	for i := range lbvserverMaps {
		lbvserverMap := lbvserverMaps[i].(map[string]interface{})

		lbvserverDetails := make(map[string]string)
		lbvserverDetails["ipaddress"] = lbvserverMap["ipv46"].(string)
		lbvserverDetails["port"] = fmt.Sprintf("%s", lbvserverMap["port"])

		name := lbvserverMap["name"].(string)
		lbvservers[name] = lbvserverDetails
	}
	return lbvservers, nil
}

func (n NitroApi) GetLbvserverBindings(lbvserverName string) (map[string]map[string]string, error) {
	resp, err := n.executeRequest("GET", "/nitro/v1/config/lbvserver_service_binding/" + lbvserverName, "", "")
	if err != nil {
		return nil, err
	}
	var jsonMap interface{}
	err = json.Unmarshal(resp.([]byte), &jsonMap)
	if err != nil {
		return nil, err
	}
	lbvserverBindings := make(map[string]map[string]string)
	lbvserverBindingMaps, ok := jsonMap.(map[string]interface{})["lbvserver_service_binding"].([]interface{})
	if !ok {
		return lbvserverBindings, nil
	}
	for i := range lbvserverBindingMaps {
		lbvserverBindingMap := lbvserverBindingMaps[i].(map[string]interface{})

		lbvserverBindingDetails := make(map[string]string)
		lbvserverBindingDetails["ipaddress"] = lbvserverBindingMap["ipv46"].(string)
		// port and servicetype?

		name := lbvserverBindingMap["servicename"].(string)
		lbvserverBindings[name] = lbvserverBindingDetails
	}
	return lbvserverBindings, nil
}

func (n NitroApi) GetSNIPs() (map[string]bool, error) {
	resp, err := n.executeRequest("GET", "/nitro/v1/config/nsip?filter=iptype:SNIP", "", "")
	if err != nil {
		return nil, err
	}
	var jsonMap interface{}
	err = json.Unmarshal(resp.([]byte), &jsonMap)
	if err != nil {
		return nil, err
	}
	// TODO: There's a lot of casting/type assertion going on here.
	// Is there an easy way to catch these errors and handle it appropriately?
	snipMaps := jsonMap.(map[string]interface{})["nsip"].([]interface{})
	snips := make(map[string]bool)
	for i := range snipMaps {
		snipMap := snipMaps[i].(map[string]interface{})
		snips[snipMap["ipaddress"].(string)] = true
	}
	return snips, nil
}

func (n NitroApi) DeleteNSIP(ip string) error {
	_, err := n.executeRequest("DELETE", "/nitro/v1/config/nsip/" + ip, "", "")
	return err
}

func (n NitroApi) AddNSIP(ip string) error {
	nsip := make(map[string]map[string]string)
	nsip["nsip"] = make(map[string]string)
	nsip["nsip"]["ipaddress"] = ip
	nsip["nsip"]["netmask"] = "255.255.0.0"
	nsip["nsip"]["type"] = "SNIP"

	data, err := json.Marshal(nsip)
	if err != nil {
		return err
	}
	_, err = n.executeRequest("POST", "/nitro/v1/config/nsip", "application/vnd.com.citrix.netscaler.nsip+json", string(data[:]))
	return err
}

// Refactor these to its own class
func (n NitroApi) CreateService(name string, ip string) error {
	service := make(map[string]map[string]string)
	service["service"] = make(map[string]string)
	service["service"]["name"] = name
	service["service"]["servicetype"] = "ANY"
	service["service"]["ip"] = ip
	service["service"]["port"] = "*"

	data, err := json.Marshal(service)
	if err != nil {
		return err
	}
	_, err = n.executeRequest("POST", "/nitro/v1/config/service", "application/vnd.com.citrix.netscaler.service+json", string(data[:]))
	return err
}

func (n NitroApi) DeleteService(name string) error {
	_, err := n.executeRequest("DELETE", "/nitro/v1/config/service/" + name, "application/vnd.com.citrix.netscaler.service+json", "")
	return err
}

func (n NitroApi) CreateLbvserver(lbvserverName string, vip string) error {
	lb := make(map[string]map[string]string)
	lb["lbvserver"] = make(map[string]string)
	lb["lbvserver"]["name"] = lbvserverName
	lb["lbvserver"]["servicetype"] = "ANY"
	lb["lbvserver"]["ipv46"] = vip
	lb["lbvserver"]["port"] = "*"

	data, err := json.Marshal(lb)
	if err != nil {
		return err
	}
	_, err = n.executeRequest("POST", "/nitro/v1/config/lbvserver", "application/vnd.com.citrix.netscaler.lbvserver+json", string(data[:]))
	return err
}

func (n NitroApi) DeleteLbvserver(name string) error {
	_, err := n.executeRequest("DELETE", "/nitro/v1/config/lbvserver/" + name, "application/vnd.com.citrix.netscaler.lbvserver+json", "")
	return err
}

func (n NitroApi) BindServiceToLbvserver(lbServiceName string, individualServiceName string) error {
	binding := make(map[string]map[string]string)
	binding["lbvserver_service_binding"] = make(map[string]string)
	binding["lbvserver_service_binding"]["name"] = lbServiceName
	binding["lbvserver_service_binding"]["servicename"] = individualServiceName

	data, err := json.Marshal(binding)
	if err != nil {
		return err
	}       
	_, err = n.executeRequest("POST", "/nitro/v1/config/lbvserver_service_binding/" + lbServiceName, "application/vnd.com.citrix.netscaler.lbvserver_service_binding+json", string(data[:]))
	return err
}

