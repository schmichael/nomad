package fingerprint

import (
	"fmt"
	"log"
	"strconv"
	"time"

	consul "github.com/hashicorp/consul/api"

	cstructs "github.com/hashicorp/nomad/client/structs"
)

const (
	consulAvailable   = "available"
	consulUnavailable = "unavailable"
)

// ConsulFingerprint is used to fingerprint for Consul
type ConsulFingerprint struct {
	logger    *log.Logger
	client    *consul.Client
	lastState string
}

// NewConsulFingerprint is used to create a Consul fingerprint
func NewConsulFingerprint(logger *log.Logger) Fingerprint {
	return &ConsulFingerprint{logger: logger, lastState: consulUnavailable}
}

func (f *ConsulFingerprint) Fingerprint(req *cstructs.FingerprintRequest, resp *cstructs.FingerprintResponse) error {
	// Only create the client once to avoid creating too many connections to
	// Consul.
	if f.client == nil {
		consulConfig, err := req.Config.ConsulConfig.ApiConfig()
		if err != nil {
			return fmt.Errorf("Failed to initialize the Consul client config: %v", err)
		}

		f.client, err = consul.NewClient(consulConfig)
		if err != nil {
			return fmt.Errorf("Failed to initialize consul client: %s", err)
		}
	}

	// We'll try to detect consul by making a query to to the agent's self API.
	// If we can't hit this URL consul is probably not running on this machine.
	info, err := f.client.Agent().Self()
	if err != nil {
		// TODO this should set consul in the response if the error is not nil

		// Print a message indicating that the Consul Agent is not available
		// anymore
		if f.lastState == consulAvailable {
			f.logger.Printf("[INFO] fingerprint.consul: consul agent is unavailable")
		}
		f.lastState = consulUnavailable
		return nil
	}

	if s, ok := info["Config"]["Server"].(bool); ok {
		resp.AddAttribute("consul.server", strconv.FormatBool(s))
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: unable to fingerprint consul.server")
	}
	if v, ok := info["Config"]["Version"].(string); ok {
		resp.AddAttribute("consul.version", v)
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: unable to fingerprint consul.version")
	}
	if r, ok := info["Config"]["Revision"].(string); ok {
		resp.AddAttribute("consul.revision", r)
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: unable to fingerprint consul.revision")
	}
	if n, ok := info["Config"]["NodeName"].(string); ok {
		resp.AddAttribute("unique.consul.name", n)
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: unable to fingerprint unique.consul.name")
	}
	if d, ok := info["Config"]["Datacenter"].(string); ok {
		resp.AddAttribute("consul.datacenter", d)
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: unable to fingerprint consul.datacenter")
	}

	attributes := resp.GetAttributes()
	if attributes["consul.datacenter"] != "" || attributes["unique.consul.name"] != "" {
		resp.AddLink("consul", fmt.Sprintf("%s.%s",
			attributes["consul.datacenter"],
			attributes["unique.consul.name"]))
	} else {
		f.logger.Printf("[WARN] fingerprint.consul: malformed Consul response prevented linking")
	}

	// If the Consul Agent was previously unavailable print a message to
	// indicate the Agent is available now
	if f.lastState == consulUnavailable {
		f.logger.Printf("[INFO] fingerprint.consul: consul agent is available")
	}
	f.lastState = consulAvailable
	return nil
}

func (f *ConsulFingerprint) Periodic() (bool, time.Duration) {
	return true, 15 * time.Second
}
