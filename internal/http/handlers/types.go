package handlers

import(
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Target string `json:"target"`
}

type CounterDataResponse struct {
	CounterEntries []*v1.CounterEntry `json:"counter_entries"`
	DirectCounterEntries []*v1.DirectCounterEntry `json:"direct_counter_entries"`
}

type TableEntriesResponse struct {
	TableEntries []*v1.TableEntry `json:"table_entries"`
}

// InstallTableEntriesRequest is the request body for POST /api/tables
type InstallTableEntriesRequest struct {
	TableEntries []*v1.TableEntry `json:"table_entries"`
}

// TableEntriesOperationResponse is the response body for POST and DELETE /api/tables
type TableEntriesOperationResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}