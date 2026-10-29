package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// TablesHandler dispatches GET, POST, and DELETE on /api/tables
func (d *Driver) TablesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		d.ReadTableEntriesHandler(w, r)
	case http.MethodPost:
		d.InstallTableEntriesHandler(w, r)
	case http.MethodDelete:
		d.DeleteTableEntriesHandler(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
	}
}

func (d *Driver) ReadTableEntriesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stream, err := d.Client.P4Client.Read(ctx, &v1.ReadRequest{
		Entities: []*v1.Entity{
			{Entity: &v1.Entity_TableEntry{TableEntry: &v1.TableEntry{}}},
		},
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}
		return
	}

	entries := []*v1.TableEntry{}
	for {
		response, err := stream.Recv()
		if err != nil {
			break
		}
		d.Log.Info("Got table response", "response", response)
		for _, entity := range response.GetEntities() {
			d.Log.Info("got table entity", "Entity", entity)
			if te := entity.GetTableEntry(); te != nil {
				entries = append(entries, te)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(TableEntriesResponse{TableEntries: entries}); err != nil {
		log.Printf("failed to encode table entries response: %v", err)
	}
}

func (d *Driver) InstallTableEntriesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := decodeTableEntries(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if len(entries) == 0 {
		writeJSONError(w, http.StatusBadRequest, "table_entries is required and must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err = d.Client.EditTableEntries(ctx, entries, v1.Update_INSERT)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to install table entries: " + err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(TableEntriesOperationResponse{Status: "ok", Count: len(entries)}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (d *Driver) DeleteTableEntriesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := decodeTableEntries(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if len(entries) == 0 {
		writeJSONError(w, http.StatusBadRequest, "table_entries is required and must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err = d.Client.EditTableEntries(ctx, entries, v1.Update_DELETE)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to delete table entries: " + err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(TableEntriesOperationResponse{Status: "ok", Count: len(entries)}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func decodeTableEntries(body io.Reader) ([]*v1.TableEntry, error) {
	var raw struct {
		TableEntries []json.RawMessage `json:"table_entries"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	entries := make([]*v1.TableEntry, 0, len(raw.TableEntries))
	for _, r := range raw.TableEntries {
		var entry v1.TableEntry
		if err := protojson.Unmarshal(r, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}
	return entries, nil
}