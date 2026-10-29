package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
)	
func (d *Driver) ProgramHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	config, err := d.Client.GetPipeline(ctx)

	d.Log.Info("Received prog response", "response", config)
	if err != nil {
		d.Log.Info("Received prog error", "err", err)
	}
	w.Header().Set("Content-Type", "application/json")

	if config != nil {
		p4i := config.Config.GetP4Info()
		marshalled, _ := protojson.Marshal(p4i)
		d.Log.Info("Retrieved p4info", "p4info", marshalled)
		d.Log.Info("Retrieved p4info", "p4info (unmarshalled)", *p4i)
		devConf := config.Config.GetP4DeviceConfig()
		d.Log.Info("Retrieved p4 dev config", "devConfig", devConf)
		d.Log.Info("Retrieved p4 dev config", "length", len(devConf))
	}

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(P4ProgramResponse{
			Status: "not_deployed",
		}); err != nil {
			log.Printf("failed to encode get program error response: %v", err)
		}
		return
	}

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
