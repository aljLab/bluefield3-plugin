package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"log"
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
			Error:  fmt.Sprintf("failed querying DPL runtime: %v", err),
		}); err != nil {
			log.Printf("failed to encode get program error response: %v", err)
		}
		return
	}

	/* if d.CurrentProgram != nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(P4ProgramResponse{
			Status:      "deployed",
			ProgramName: d.CurrentProgram.ProgramName,
			Tables:      api2.GetTableMetadata(d.CurrentProgram),
			Counters:    api2.GetCounterMetadata(d.CurrentProgram),
		}); err != nil {
			log.Printf("failed to encode get program response: %v", err)
		}
		return
	} */

	if config != nil && config.Config != nil && len(config.Config.P4DeviceConfig) > 0 {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(P4ProgramResponse{
			Status: "deployed",
			Error:  "program metadata not available in driver memory",
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
