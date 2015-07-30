package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"
	log "github.com/Sirupsen/logrus"

	"github.com/sonchang/nme-controller/metadata"
	"github.com/sonchang/nme-controller/nme"
)

const (
	linkLocalSNIP  = "169.254.0.100"
	maxRetriesToGetRancherIpForNME = 10
	waitMillisToGetRancherIpForNME = 1000
)

var (
	debug          = flag.Bool("debug", true, "Debug")
	nmeContainerId = flag.String("nmeContainerId", "", "nme container ID")
	metadataUrl    = flag.String("metadata", "metadata:8083", "URL to metadata server")
	nmeRestUrl     = flag.String("nme", "", "URL to Netscaler NITRO REST API")
	poll           = flag.Int("poll", 1000, "Poll interval in millis")
)

func main() {
	log.Info("Starting Netscaler controller")
	parseFlags()

	metadataHandler := metadata.NewHandler(*metadataUrl)
	nmeHandler := nme.NewHandler(nme.NewNitroApi(*nmeRestUrl))

	initializeNME(nmeHandler)

	// TODO: query nme to obtain lbconfigs in case of server restart
	lbConfig := new(nme.LbConfigs)

	for {
		hash, err := metadataHandler.GetHash()
		if err != nil {
			log.Errorf("error = %v", err)
			time.Sleep(time.Duration(*poll) * time.Millisecond)
			continue;
		}
		if hash == lbConfig.Hash {
			log.Debugf("no change in hash: %s", hash)
			time.Sleep(time.Duration(*poll) * time.Millisecond)
			continue
		}
		newConfig, err := metadataHandler.GetLbConfig()
		if newConfig != nil {
			log.Debugf("newConfig = %v", newConfig)
			err = applyDiffs(lbConfig.LbConfig, newConfig, nmeHandler)
		}
		if err != nil {
			log.Errorf("error = %v", err)
		} else {
			lbConfig.Hash = hash
			lbConfig.LbConfig = newConfig
		}

		time.Sleep(time.Duration(*poll) * time.Millisecond)
	}
}

func initializeNME(nmeHandler nme.NmeHandler) error {
	err := nmeHandler.AddNSIP(linkLocalSNIP)
	if err != nil {
		return err
	}

	// get rancher's managed network IP for nme container
	cmdstr := fmt.Sprint("/usr/bin/docker exec ", *nmeContainerId, " ip addr show | grep -oP \"10\\.42\\.(\\d+)\\.(\\d+)\"")
	attempts := 1
	for {
		cmd := exec.Command("bash", "-c", cmdstr)
		rancherIp, err := cmd.Output()
		if err != nil {
			if attempts > maxRetriesToGetRancherIpForNME {
				log.Fatalf("error = %v", err)
			}
			log.Errorf("attempt %v: error = %v", attempts, err)
			time.Sleep(time.Duration(waitMillisToGetRancherIpForNME) * time.Millisecond)
			attempts++
		} else {
			return nmeHandler.AddNSIP(strings.TrimSpace(string(rancherIp)))
		}
	}
}


func applyDiffs(currentConfig map[string]nme.Lbvserver, newConfig map[string]nme.Lbvserver, nmeHandler nme.NmeHandler) error {
	for lbvserverName, newLbvserver := range newConfig {
		// add any new services
		if currentLbvserver, ok := currentConfig[lbvserverName]; !ok {
			err := nmeHandler.CreateLB(newLbvserver)
			if err != nil {
				return err
			}
		} else {
			// update existing lbvserver with new service bindings
			// ASSUMPTION: VIP does not change
			for serviceName, newService := range newLbvserver.Bindings {
				var err error
				if currentService, ok := currentLbvserver.Bindings[serviceName]; !ok {
					// create new service+binding
					err = nmeHandler.CreateServiceAndBinding(newLbvserver, newService)
				} else {
					// check whether IP needs to be updated or not
					if newService.IpAddress != currentService.IpAddress {
						err = nmeHandler.UpdateServiceIp(newLbvserver, newService)
					}
				}
				if err != nil {
					return err
				}
			}
			for serviceName, currentService := range currentLbvserver.Bindings {
				if _, ok := newLbvserver.Bindings[serviceName]; !ok {
					err := nmeHandler.DeleteServiceAndBinding(currentService)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// remove deleted services
	for lbvserverName, currentLbvserver := range currentConfig {
		if _, ok := newConfig[lbvserverName]; !ok {
			err := nmeHandler.DeleteLB(currentLbvserver)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseFlags() {
	flag.Parse()

	log.Debugf("nmeContainerId=%s, metadataUrl=%s, nmeUrl=%s", *nmeContainerId, *metadataUrl, *nmeRestUrl)

	if *debug {
		log.SetLevel(log.DebugLevel)
	}
}