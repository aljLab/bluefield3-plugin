package networking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/go-logr/logr"

)
const (

)

func ConfigureNetwork(setupLog logr.Logger, targetName string) error {
	// get driver IP from control interface eth0 (or see in deployment args) -> flannel endpoint in pod namespace
	setupLog.Info("Looking up driver IP for control interface assigned by primary CNI.")
	// get netConfig with: ClusterCIDR, ServiceCIDR, (host ns) iface name , (host ns) bridge name
	setupLog.Info("Requesting P4target configuration from IML daemon...")
	
	req := P4TargetConfigRequest{
		TargetName: targetName,
	}
	// would be nice to get the namespace as well
	netConf, err := getP4TargetConfigFromIML(req); if err != nil {
		return fmt.Errorf("while getting p4target config: %w", err)
	}
	
	fmt.Printf("Netconf from IML: {bridge: %s, ClusterCidrs: %s, Iface: %s, MTU: %d}", netConf.BridgeName, netConf.ClusterCIDRs, netConf.IfaceName, netConf.MTU)
	
	// let operator assign NfCIDR
	setupLog.Info("Checking if NfCIDR has been assigned by IML operator...")

	setupLog.Info("Deploying network configuration...")
	// get pod namespace

	// move bluefield interfaces into the pod ns
	// create pod networking (veth <-> routingBridge, NfVrf, br0, routing)
	
	// configure proxy networking
		// rule for sidnet and nfcidr through tun0
		// install routes/rules
		// tun devices/interfaces are created by proxy -> configure names here
	return nil
}

func getP4TargetConfigFromIML(configRequest P4TargetConfigRequest) (*NetworkConfig, error) {
	data, err := json.Marshal(configRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	resp, err := http.Post(
		"http://localhost:7623/api/v1/cni/p4target/register",
		"application/json", bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-2XX response: %s", resp.Status)
	}
	var configResponse NetworkConfig
	if err = json.NewDecoder(resp.Body).Decode(&configResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &configResponse, nil
}
