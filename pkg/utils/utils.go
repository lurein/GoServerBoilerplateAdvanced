package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"
	"whimsy/pkg/constants"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var santizeFields = []string{
}

func StrPtr(s string) *string {
	return &s
}

// santizeRegexFields is a cached regex.Compile called for each field.
var santizeRegexFields = func() map[string]*regexp.Regexp {
	m := make(map[string]*regexp.Regexp)
	for _, field := range santizeFields {
		// Case-insensitive regex match
		regexstr := fmt.Sprintf(`(?i)("%s"):\s?"([^\"]+)"`, field)
		m[field] = regexp.MustCompile(regexstr)
	}
	return m
}()

func Message(status bool, message string) map[string]interface{} {
	return map[string]interface{}{"status": status, "message": message}
}

func Respond(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func ReportError(ctx context.Context, err error) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
	} else {
		sentry.CaptureException(err)
	}
}

func LogAndReportError(ctx context.Context, err error, msg string) {
	logger := zerolog.Ctx(ctx)
	logger.Err(err).Msg(msg)
	ReportError(ctx, err)
}

// This type implements the http.RoundTripper interface
type HttpLoggingRoundTripper struct {
	Proxied http.RoundTripper
	Label   string
}

func redactedStr(input string) string {
	return strings.Repeat("*", len(input))
}

func SanitizeDump(input string) string {
	for _, field := range santizeFields {
		reg := santizeRegexFields[field]
		rs := reg.FindStringSubmatch(input)

		if len(rs) > 1 {
			replacementStr := fmt.Sprintf(`$1:"%s"`, redactedStr(rs[2]))
			input = reg.ReplaceAllString(input, replacementStr)
		}
	}

	return input
}

func GetStringValueFromContext(key constants.ContextKey, ctx context.Context) string {
	value, _ := ctx.Value(key).(string)
	return value
}

func (lrt HttpLoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	logger := zerolog.Ctx(ctx)

	startTime := time.Now()

	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		logger.Err(err).Send()
		return nil, err
	}

	userId := ""
	contextval := ctx.Value(constants.UserIDKey)
	if contextval != nil {
		userId = contextval.(string)
	}

	logger.Trace().
		Str("provider", lrt.Label).
		Str("request_body", SanitizeDump(string(dump))).
		Str("user_id", userId).
		Send()

	res, err := lrt.Proxied.RoundTrip(req)
	if err != nil {
		logger.Error().
			Err(err).
			Str("method", req.Method).
			Str("provider", lrt.Label).
			Str("path", req.URL.Path).
			Dur("duration", time.Since(startTime)).
			Str("user_id", userId).
			Send()
		return nil, err
	}

	if dump, err = httputil.DumpResponse(res, true); err != nil {
		logger.Error().Err(err).Send()
	} else {
		logger.Trace().
			Str("response_body", SanitizeDump(string(dump))).
			Str("provider", lrt.Label).
			Int("code", res.StatusCode).
			Str("user_id", userId).
			Send()
	}

	return res, nil
}

func CreateTaggedLogger(action string, ctx context.Context) *zerolog.Logger {
	userID := GetStringValueFromContext(constants.UserIDKey, ctx)
	userReferenceID := GetStringValueFromContext(constants.UserReferenceIDKey, ctx)

	logger := zerolog.Ctx(ctx)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		if userID != "" {
			c = c.Str("user_id", userID)
		}
		if userReferenceID != "" {
			c = c.Str("user_reference_id", userReferenceID)
		}
		return c.Str("action", action)
	})
	return logger
}

func DoesArrayContainString(array []string, str string) bool {
	for _, a := range array {
		if a == str {
			return true
		}
	}
	return false
}

func GetDeviceOS(r *http.Request) string { // since this field is populated by FE, it will always be iOS or Android
	platform := r.Header.Get("X-Platform")
	return platform
}
