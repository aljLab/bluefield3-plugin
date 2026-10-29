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
}