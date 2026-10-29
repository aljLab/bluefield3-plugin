package handlers

import(
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (d *Driver) DevicesHandler(w http.ResponseWriter, r *http.Request){
	url := fmt.Sprintf("%s/dpl/devices", d.DplServerAddr)

	w.Header().Set("Content-Type", "application/json")

	res, err := http.Get(url); if err != nil {
		d.Log.Info("Got timed out or redirected too often in devices request.", "DPL server", d.DplServerAddr)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(
			ErrorResponse{
				Error: fmt.Sprintf("Could not reach DPL server on %s. Is the server running?", d.DplServerAddr),
			},
		)

		return
	}

	if !strings.HasPrefix(res.Status, "2"){
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(
			ErrorResponse{
				Error: "Received invalid response status from nspect server",
			},
		)
		return
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		d.Log.Info("Failed to read response body from nspect devices")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(
			ErrorResponse{
				Error: "Could read response body from nspect devices.",
			},
		)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
