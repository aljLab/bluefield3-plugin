package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)	
func (d *Driver) ProgramHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	config, err := d.Client.GetPipeline(ctx)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(P4ProgramResponse{
			Status: "not_deployed",
		}); err != nil {
			log.Printf("failed to encode get program error response: %v", err)
		}
		return
	}

	// DPL does not return the device config binary through p4runtime -> relying on presence of p4info
	if config != nil && config.Config != nil && config.Config.P4Info != nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(P4ProgramResponse{
			Status: "deployed",
			P4Info:  config.Config.P4Info,
		}); err != nil {
			log.Printf("failed to encode get program deployed response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(P4ProgramResponse{Status: "not_deployed"}); err != nil {
		log.Printf("failed to encode get program not deployed response: %v", err)
	}
}
