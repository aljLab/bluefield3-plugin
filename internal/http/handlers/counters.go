package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (d *Driver) ReadCountersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	/*
	READ COUNTERS
	*/
	reqCounters := &v1.ReadRequest{
		DeviceId: d.DeviceId,
		Entities: []*v1.Entity{
			{Entity: &v1.Entity_CounterEntry{CounterEntry: &v1.CounterEntry{
				CounterId: 0, // 
			}}},
		},
	}

	stream, err := d.Client.P4Client.Read(ctx, reqCounters)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}
		return
	}

	var counterEntries []*v1.CounterEntry
	for {
		response, err := stream.Recv()
		d.Log.Info("Received counter resp", "Response", response)
		if err != nil {
			break
		}
		for _, entity := range response.GetEntities() {
			ce := entity.GetCounterEntry()
			if ce != nil && ce.Data != nil && (ce.Data.PacketCount != 0 || ce.Data.ByteCount != 0) {
				counterEntries = append(counterEntries, ce)
			}
		}
	}

	/*
	READ DIRECT COUNTER ENTRIES
	*/

	reqDirectCounters := &v1.ReadRequest{
		DeviceId: d.DeviceId,
		Entities: []*v1.Entity{
			{Entity: &v1.Entity_DirectCounterEntry{DirectCounterEntry: &v1.DirectCounterEntry{
				TableEntry: &v1.TableEntry{ TableId: 0 },

			}}},
		},
	}

	stream, err = d.Client.P4Client.Read(ctx, reqDirectCounters)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}
		return
	}

	var directCounterEntries []*v1.DirectCounterEntry
	for {
		response, err := stream.Recv()
		d.Log.Info("Received direct counter resp", "Response", response)
		if err != nil {
			break
		}
		for _, entity := range response.GetEntities() {
			ce := entity.GetDirectCounterEntry()
			if ce != nil && ce.Data != nil && (ce.Data.PacketCount != 0 || ce.Data.ByteCount != 0) {
				directCounterEntries = append(directCounterEntries, ce)
			}
		}
	}

	d.Log.Info("Fetched counter entries: ", "Len", len(directCounterEntries))

	res := CounterDataResponse{
		CounterEntries: counterEntries,
		DirectCounterEntries: directCounterEntries,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("failed to encode counter data response: %v", err)
	}
}