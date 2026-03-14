package response

import "github.com/redhajuanda/komon/pagination"

// Metadata struct
type Metadata struct {
	RequestID     string             `json:"request_id"`
	CorrelationID string             `json:"correlation_id"`
	Pagination    *pagination.Result `json:"pagination,omitempty"`
}