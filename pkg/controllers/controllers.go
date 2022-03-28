package controllers

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"

	"whimsy/pkg/constants"
	"whimsy/pkg/errors"
	"whimsy/pkg/utils"


	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

// Controller is a http.Handler with routes.
type Controller interface {
	// Route implements gorilla mux routing.
	Route(*mux.Router)
}

var RequestLimit int64 = 10000

func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	utils.Respond(w, utils.Message(true, "OK"))
}

func Welcome(w http.ResponseWriter, _ *http.Request) {
	utils.Respond(w, utils.Message(true, "Welcome to Sycamore"))
}

func NotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	utils.Respond(w, utils.Message(false, "This route was not found on our server"))
}

// APIHandler wraps an error handler for typed error responses.
func APIHandler(errHandler func(w http.ResponseWriter, r *http.Request) error, obfuscateError bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := errHandler(w, r); err != nil {
			ctx := r.Context()
			logger := zerolog.Ctx(ctx)

			var friendlyErr *errors.Error
			if !goerrors.As(err, &friendlyErr) {
				// Capture private error messages and report generic.
				logger.Err(err).Msg("unhandled api handler error")
				friendlyErr = errors.NewGenericError(err)
			} else {
				logger.Debug().Err(friendlyErr).Msg("api handler error")
			}

			// for some endpoints like /admin we always want to return
			// the raw error
			var returnedError interface{}
			if obfuscateError {
				returnedError = friendlyErr
			} else {
				returnedError = err
			}

			if err := writeBody(w, returnedError); err != nil {
				utils.LogAndReportError(ctx, err, "failed to encode error response")
			}
		}
	})
}

func readBody(r *http.Request, out interface{}) (err error) {
	defer func() {
		cerr := r.Body.Close()
		if err == nil {
			err = cerr
		}
	}()
	if err := json.NewDecoder(io.LimitReader(r.Body, RequestLimit)).Decode(out); err != nil {
		fmt.Println(err)
		return errors.NewInvalidRequestBodyFormatError()
	}
	return nil
}

func writeBody(w http.ResponseWriter, body interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err, ok := body.(error); ok {
		w.WriteHeader(errors.StatusCode(err))
	}
	return json.NewEncoder(w).Encode(body)
}

func setUserIdInContext(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, constants.UserIDKey, userId)
}