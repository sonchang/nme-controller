package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	log "github.com/Sirupsen/logrus"
)

var (
	debug       = flag.Bool("debug", true, "Debug")
	metadataUrl = flag.String("metadata", "metadata:8083", "URL to metadata server")
	nmeRestUrl  = flag.String("nme", "", "URL to Netscaler NITRO REST API")
	poll        = flag.Int("poll", 1000, "Poll interval in millis")

        metadataHash string
	lbConfig     []interface{}
)

func main() {
	log.Info("Starting Netscaler controller")
	parseFlags()
	for {
		hash, err := getMetadataHash()
		if err != nil {
			log.Errorf("error = %v", err)
			time.Sleep(time.Duration(*poll) * time.Millisecond)
			continue;
		}
		if hash == metadataHash {
			time.Sleep(time.Duration(*poll) * time.Millisecond)
			continue
		}
		metadataHash = hash
		// TODO: Figure out diff
		mappings, err := getMappingsFromMetadata()
		if err != nil {
			log.Errorf("error = %v", err)
		} else {
			log.Debugf("mappings = %v", mappings)
			err = applyMappingsToLB(mappings)
		}

		time.Sleep(time.Duration(*poll) * time.Millisecond)
	}
}

func getMetadataHash() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + *metadataUrl + "/latest/hash", nil)
        resp, err := client.Do(req)
        defer resp.Body.Close()
        if err != nil {
                return "", err
        }
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return "", err
        }
        log.Debugf("resp body=%s", string(body[:]))
	return string(body[:]), nil
}

// TODO: Possibly create a struct type for mappings rather than generic interface{}
// Make HTTP GET from metadata, parse JSON results, and map to LB mappings
func getMappingsFromMetadata() (interface{}, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + *metadataUrl + "/latest/stacks", nil)
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
	return mappings, err
}

// NOTE: Since this might change, I'm not spending too much time
// making this clean
func applyMappingsToLB(mappings interface{}) error {
	if mappings == nil {
		// TODO: handle diff
		return nil
	}
	stacks, ok := mappings.([]interface{})
	if ok {
		for i := range stacks {
			stack := stacks[i]
			stackMetadata, ok := stack.(map[string]interface{})
			if !ok {
				continue
			}
			services, ok := stackMetadata["services"].([]interface{})
			if services == nil || !ok {
				continue
			}
			for j := range services {
				service, ok := services[j].(map[string]interface{})
				if !ok {
					continue
				}
				vip := service["ip"].(string)
				serviceName := service["name"].(string)
				containers := service["containers"].([]interface{})
				createLB(vip, serviceName, containers)
			}
		}
	} else {
		return fmt.Errorf("Error parsing services from metadata server %v", mappings)
	}
	return nil
}

func postToNitro(url string, contentType string, jsonContent string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://" + *nmeRestUrl + "/" + url, nil)
	req.Header.Add("X-NITRO-USER", "root")
	req.Header.Add("X-NITRO-PASS", "linux")
	req.Header.Add("Content-Type", contentType)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	// TODO: Error handling
	return err
}

func createLB(vip string, serviceName string, containers []interface{}) error {
	err := createLbvserver(serviceName, vip)
	if err != nil {
		return err
	}
	for i := range containers {
		container := containers[i].(map[string]string)
		name := container["name"]
		ip := container["ip"]
		err := createService(name, ip)
		if err != nil {
			return err
		}
		err = bindServiceToLbvserver(serviceName, name)
		if err != nil {
			return err
		}
	}
	return nil
}

// Refactor these to its own class
func createService(name string, ip string) error {
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
	err = postToNitro("/nitro/v1/config/service", "application/vnd.com.citrix.netscaler.service+json", string(data[:]))
	return err
}

func createLbvserver(serviceName string, vip string) error {
	lb := make(map[string]map[string]string)
	lb["lbvserver"] = make(map[string]string)
	lb["lbvserver"]["name"] = serviceName
	lb["lbvserver"]["servicetype"] = "ANY"
	lb["lbvserver"]["ipv46"] = vip
	lb["lbvserver"]["port"] = "*"

	data, err := json.Marshal(lb)
	if err != nil {
		return err
	}
	err = postToNitro("/nitro/v1/config/lbvserver", "application/vnd.com.citrix.netscaler.lbvserver+json", string(data[:]))
        return err
}

func bindServiceToLbvserver(lbServiceName string, individualServiceName string) error {
	binding := make(map[string]map[string]string)
	binding["lbserver_service_binding"] = make(map[string]string)
	binding["lbserver_service_binding"]["name"] = lbServiceName
	binding["lbserver_service_binding"]["servicename"] = individualServiceName

        data, err := json.Marshal(binding)
        if err != nil {
                return err
        }       
        err = postToNitro("/nitro/v1/config/lbvserver_service_binding/" + lbServiceName, "application/vnd.com.citrix.netscaler.lbvserver_service_binding+json", string(data[:]))
        return err
}

func parseFlags() {
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}
}
