package cmsclient

import (
	"encoding/json"
	"fmt"
)

// APIError represents a failed CMS response envelope.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Code + ": " + e.Message
}

// Envelope is the successful portion of the CMS response contract. Data and
// Meta remain raw so consumers can use their own endpoint-specific DTOs.
type Envelope struct {
	Data json.RawMessage
	Meta json.RawMessage
}

// DecodeEnvelope validates and unwraps the standard CMS API response.
func DecodeEnvelope(body []byte) (Envelope, error) {
	var wire struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Meta    json.RawMessage `json:"meta"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return Envelope{}, fmt.Errorf("decode CMS response: %w", err)
	}
	if !wire.Success {
		if wire.Error == nil {
			return Envelope{}, &APIError{Code: "UNKNOWN", Message: "CMS request failed"}
		}
		return Envelope{}, &APIError{Code: wire.Error.Code, Message: wire.Error.Message}
	}
	return Envelope{Data: wire.Data, Meta: wire.Meta}, nil
}
