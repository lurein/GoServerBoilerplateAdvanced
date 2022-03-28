package errors

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/hlog"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Messages are based on google API error messages.
// https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto

// A message type used to describe a single bad request field.
type FieldViolation struct {
	// A path leading to a field in the request body. The value will be a
	// sequence of dot-separated identifiers that identify a mesasge field.
	// E.g., "fieldViolations.field" would identify this field.
	Field string `json:"field,omitempty"`

	// A description of why the request element is bad.
	Description string `json:"description,omitempty"`
}

func CreateFieldViolation(field string, format string, a ...interface{}) FieldViolation {
	return FieldViolation{
		Field:       field,
		Description: fmt.Sprintf(format, a...),
	}
}

// Describes violations in a client request. This error type focuses on the
// syntactic aspects of the request.
type BadRequest struct {
	// Describes all violations in a client request.
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

func NewBadRequest(violations ...FieldViolation) *BadRequest {
	return &BadRequest{
		FieldViolations: violations,
	}
}

func (v *BadRequest) WithFieldViolation(field string, format string, a ...interface{}) *BadRequest {
	v.FieldViolations = append(v.FieldViolations, CreateFieldViolation(field, format, a...))
	return v
}

// Contains metadata about the request that clients can attach when filing a bug
// or providing other forms of feedback.
type RequestInfo struct {
	// An opaque string that should only be interpreted by the service generating
	// it. For example, it can be used to identify requests in the service's logs.
	RequestID string `json:"requestID,omitempty"`

	// Any data that was used to serve this request. For example, an encrypted
	// stack trace that can be sent back to the service provider for debugging.
	ServingData string `json:"servingData,omitempty"`
}

// NewRequestInfo logs data about the Request.ID stored in the context.
func NewRequestInfo(ctx context.Context, _ int) *RequestInfo {
	v := &RequestInfo{}
	if id, ok := hlog.IDFromCtx(ctx); ok {
		v.RequestID = id.String()
	}
	// TODO: secure encoding, if wanted.
	//// get current filename and linenumber
	//if _, filename, line, ok := runtime.Caller(depth+1); ok {
	//	v.ServingData = fmt.Sprintf("%s:%d", filename, line)
	//}
	return v
}

// Provides a localized error message that is safe to return to the user
// which can be attached to an RPC error.
type LocalizedMessage struct {
	// The locale used following the specification defined at
	// http://www.rfc-editor.org/rfc/bcp/bcp47.txt.
	// Examples are: "en-US", "fr-CH", "es-MX"
	Locale string `json:"locale,omitempty"`

	// The localized error message in the above locale.
	Message string `json:"message,omitempty"`
}

func NewLocalizedMessage(tag language.Tag, key message.Reference, a ...interface{}) *LocalizedMessage {
	p := message.NewPrinter(tag)
	msg := p.Sprintf(key, a...)
	return &LocalizedMessage{
		Locale:  tag.String(),
		Message: msg,
	}
}

// swagger:enum ReasonType
type ReasonType string

const (
	ReasonUnknown         ReasonType = "UNKNOWN"
	ReasonOutdatedVersion ReasonType = "OUTDATED_VERSION"
)

// Example of an error with outdated client version:
//
//     { "reason": "OUTDATED_VERSION"
//       "metadata": {
//         "version": "v0.0.1",
//         "minimum_version": "v0.0.2"
//       }
//     }
//

// Describes the cause of the error with structured details.
type ErrorInfo struct {
	// The reason of the error. This is a constant value that identifies the
	// proximate cause of the error.
	// This should be at most 63 characters and match /[A-Z0-9_]+/.
	Reason   ReasonType
	Metadata map[string]string
}

func NewErrorInfo(reason ReasonType, metadata map[string]string) *ErrorInfo {
	return &ErrorInfo{
		Reason:   reason,
		Metadata: metadata,
	}
}
