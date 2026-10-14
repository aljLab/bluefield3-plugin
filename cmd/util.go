package main

import(
	"bytes"
	"fmt"
	"net/http"
	"encoding/json"
)

func getP4TargetConfigFromIML(configRequest types2.ContainerizedP4TargetConfigRequest) (*types2.NetworkConfig, error) {
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
	var configResponse types2.NetworkConfig
	if err = json.NewDecoder(resp.Body).Decode(&configResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &configResponse, nil
}