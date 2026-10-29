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

// MatchFieldMetadata describes a match field in a P4 table
type MatchFieldMetadata struct {
	FieldID   uint32 `json:"field_id"`
	FieldName string `json:"field_name"`
	MatchType string `json:"match_type"`
	Bitwidth  int32  `json:"bitwidth"`
}

// ActionParamMetadata describes a parameter of a P4 action
type ActionParamMetadata struct {
	ParamID  uint32 `json:"param_id"`
	Name     string `json:"name"`
	Bitwidth int32  `json:"bitwidth"`
}

// ActionMetadata describes a P4 action with its ID and parameters
type ActionMetadata struct {
	ActionID   uint32                `json:"action_id"`
	ActionName string                `json:"action_name"`
	Params     []ActionParamMetadata `json:"params"`
}

// TableMetadata contains information about a P4 table
type TableMetadata struct {
	TableID     uint32               `json:"table_id"`
	TableName   string               `json:"table_name"`
	Size        uint32               `json:"size"`
	MatchFields []MatchFieldMetadata `json:"match_fields"`
	Actions     []ActionMetadata     `json:"actions"`
}

// CounterMetadata contains information about a P4 counter
type CounterMetadata struct {
	CounterID   uint32 `json:"counter_id"`
	CounterName string `json:"counter_name"`
	Unit        string `json:"unit"`
}


type P4ProgramResponse struct {
	Status      string            `json:"status"`          // "deployed" or "not_deployed"
	ProgramName string            `json:"program_name"`    // Name of deployed program
	Tables      []TableMetadata   `json:"tables"`          // Table metadata
	Counters    []CounterMetadata `json:"counters"`        // Counter metadata
	Error       string            `json:"error,omitempty"` // Error message if any
}