package errors

import (
	"context"
	goerrors "errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"golang.org/x/text/language"
)

// Error default error structure
// swagger:model WhimsyError
type Error struct {
	error `json:"-"`

	// required: true
	Msg              string            `json:"msg"`
	FieldErrors      map[string]string `json:"fieldErrors"`
	HTTPStatus       int               `json:"code,omitempty"`
	BadRequest       *BadRequest       `json:"badRequest,omitempty"`
	RequestInfo      *RequestInfo      `json:"requestInfo,omitempty"`
	LocalizedMessage *LocalizedMessage `json:"localizedMessage,omitempty"`
	ErrorInfo        *ErrorInfo        `json:"errorInfo,omitempty"`
}

// WhimsyErrorResponse swagger object
//
// swagger:response
type WhimsyErrorResponse struct {
	// in:body
	Body Error
}

func (e *Error) Error() string {
	var sb strings.Builder
	sb.WriteString(http.StatusText(e.HTTPStatus))
	sb.WriteString(": ")
	sb.WriteString(e.Msg)

	if br := e.BadRequest; br != nil {
		sb.WriteString(": FieldViolations(")
		for i, fv := range br.FieldViolations {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fv.Field)
			sb.WriteString(":")
			sb.WriteString(fv.Description)
		}
		sb.WriteString(")")
	}
	if ri := e.RequestInfo; ri != nil && ri.RequestID != "" {
		sb.WriteString(": RequestInfo(")
		sb.WriteString("requestID:")
		sb.WriteString(ri.RequestID)
		sb.WriteString(")")
	}
	if ei := e.ErrorInfo; ei != nil {
		sb.WriteString(": ErrorInfo(")
		sb.WriteString(string(ei.Reason))
		var i int
		for key, val := range ei.Metadata {
			if i == 0 {
				sb.WriteString(",Metadata:{")
			} else {
				sb.WriteString(",")
			}
			sb.WriteString(key)
			sb.WriteString(":")
			sb.WriteString(val)
			i++
		}
		if i > 0 {
			sb.WriteString("}")
		}
	}
	return sb.String()
}
func (e *Error) Unwrap() error { return e.error }

func NewInvalidRequestBodyFormatError() *Error {
	return &Error{Msg: "Failed to parse request body.", HTTPStatus: http.StatusBadRequest}
}
func NewBadRequestError(err error) (e *Error) {
	if goerrors.As(err, &e) {
		return e
	}

	e = &Error{
		error:      err,
		Msg:        "Bad request.",
		HTTPStatus: http.StatusBadRequest,
	}

	var ves validator.ValidationErrors
	if goerrors.As(err, &ves) {
		e.WithValidationErrors(ves)
	}
	return e

}
func NewBadRequestErrorWithMessage(msg string) *Error {
	return &Error{Msg: msg, HTTPStatus: http.StatusBadRequest}
}
func NewGenericError(err error) *Error {
	var e *Error
	if goerrors.As(err, &e) {
		return e
	}
	return &Error{error: err, Msg: "Internal server error.", HTTPStatus: http.StatusInternalServerError}
}
func NotFoundError() *Error {
	return &Error{Msg: "not found", HTTPStatus: http.StatusNotFound}
}

func NewErrorf(ctx context.Context, status int, format string, a ...interface{}) *Error {
	requestInfo := NewRequestInfo(ctx, 1)
	localizedMessage := NewLocalizedMessage(language.AmericanEnglish, format, a...) // TODO: needed?
	return (&Error{
		Msg:              fmt.Sprintf(format, a...),
		HTTPStatus:       status,
		RequestInfo:      requestInfo,
		LocalizedMessage: localizedMessage,
	})
}

func WrapErrorf(ctx context.Context, err error, status int, format string, a ...interface{}) (e *Error) {
	// If any error in the chain is a sycamore error, return the original error.
	if goerrors.As(err, &e) {
		return e
	}

	requestInfo := NewRequestInfo(ctx, 1)
	localizedMessage := NewLocalizedMessage(language.AmericanEnglish, format, a...) // TODO: needed?
	e = &Error{
		error:            err,
		Msg:              fmt.Sprintf(format, a...),
		HTTPStatus:       status,
		RequestInfo:      requestInfo,
		LocalizedMessage: localizedMessage,
	}

	var ves validator.ValidationErrors
	if goerrors.As(err, &ves) {
		e.WithValidationErrors(ves)
	}
	return e
}

// formatAttribute from User.Title.CasedID to title.casedID
func formatAttribute(input string) string {
	input = strings.TrimPrefix(input, "User.")
	rs := make([]rune, 0, len(input))
	toLower := true
	for _, r := range input {
		if toLower {
			r = unicode.ToLower(r)
		}
		toLower = r == '.'
		rs = append(rs, r)
	}
	return string(rs)
}

// TODO: better testing.
var debugFieldViolations bool = false

func parseFieldViolations(ves validator.ValidationErrors) []FieldViolation {
	fieldViolations := make([]FieldViolation, len(ves))
	for i, fe := range ves {
		field := formatAttribute(fe.Namespace())
		desc := fe.Error() // default

		switch fe.Tag() {
		case "gte":
			desc = fmt.Sprintf("Must be greater than or equal to %v.", fe.Value())
		case "gt":
			desc = fmt.Sprintf("Must be greater than to %v.", fe.Value())
		case "lte":
			desc = fmt.Sprintf("Must be less than or equal to %v.", fe.Value())
		case "lt":
			desc = fmt.Sprintf("Must be less than %v.", fe.Value())
		case "required", "required_unless", "required_with":
			desc = "Required."
		case "not_po_box":
			desc = "P.O. Box not supported."
		}

		if debugFieldViolations {
			fmt.Println("NS", fe.Namespace())
			fmt.Println("FIELD", fe.Field())
			fmt.Println("STRUCT_NS", fe.StructNamespace())
			fmt.Println("STRUCT_FIELD", fe.StructField())
			fmt.Println("TAG", fe.Tag())
			fmt.Println("ACTUAL_TAG", fe.ActualTag())
			fmt.Println("KIND", fe.Kind())
			fmt.Println("TYPE", fe.Type())
			fmt.Println("VALUE", fe.Value())
			fmt.Println("PARAM", fe.Param())
			fmt.Println("ERROR", fe.Error())
			fmt.Println("DESC", desc)
			fmt.Println()
		}

		fieldViolations[i] = FieldViolation{
			Field:       field,
			Description: desc,
		}
	}
	return fieldViolations
}

// TODO: remove in favour of BadRequest.FieldViolations
func parseFieldErrors(ves validator.ValidationErrors) map[string]string {
	fieldErrors := make(map[string]string, len(ves))
	for _, fe := range ves {
		msg := "invalid"

		// Handle specific error messages here.
		switch fe.Tag() {
		case "not_po_box":
			msg = "We don’t accept P.O. boxes"
		}
		fieldErrors[formatAttribute(fe.Namespace())] = msg
	}
	return fieldErrors
}

// WithFieldViolation to add a custom field to an error. The description should be presentable to the user.
func (e *Error) WithFieldViolation(field string, format string, a ...interface{}) {
	if e.BadRequest == nil {
		e.BadRequest = &BadRequest{}
	}
	e.BadRequest.WithFieldViolation(field, format, a...)

	if e.FieldErrors == nil {
		e.FieldErrors = make(map[string]string)
	}
	e.FieldErrors[field] = "invalid"
}

// WithValidationError adds all the validator.ValidationErrors field errors.
func (e *Error) WithValidationErrors(ves validator.ValidationErrors) {
	if len(ves) == 0 {
		return
	}

	fieldViolations := parseFieldViolations(ves)
	if e.BadRequest == nil {
		e.BadRequest = &BadRequest{
			FieldViolations: fieldViolations,
		}
	} else {
		e.BadRequest.FieldViolations = append(e.BadRequest.FieldViolations, fieldViolations...)
	}

	if e.FieldErrors == nil {
		e.FieldErrors = make(map[string]string)
	}

	for key, val := range parseFieldErrors(ves) {
		e.FieldErrors[key] = val
	}
}

func (e *Error) WithError(err error) {
	if e.error == nil {
		e.error = err
	}

	var e2 *Error
	if goerrors.As(err, &e2) {
		if e2.FieldErrors != nil {
			if e.FieldErrors == nil {
				e.FieldErrors = make(map[string]string)
			}

			for key, val := range e2.FieldErrors {
				e.FieldErrors[key] = val
			}
		}

		if e2.BadRequest != nil {
			if e.BadRequest == nil {
				e.BadRequest = &BadRequest{}
			}
			e.BadRequest.FieldViolations = append(
				e.BadRequest.FieldViolations,
				e2.BadRequest.FieldViolations...,
			)
		}

		// TODO: request, message?

		return
	}

	var ves validator.ValidationErrors
	if goerrors.As(err, &ves) {
		e.WithValidationErrors(ves)
	}

	// TODO: gorm errors?
}

func (e *Error) WithReason(reason ReasonType, metadata map[string]string) {
	e.ErrorInfo = NewErrorInfo(reason, metadata)
}

func StatusCode(err error) int {
	var e *Error
	if goerrors.As(err, &e) {
		return e.HTTPStatus
	}
	return http.StatusInternalServerError
}
