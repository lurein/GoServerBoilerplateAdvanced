package errors

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/rs/zerolog/hlog"
)

func TestFieldFormat(t *testing.T) {
	want := "title.casedID"
	got := formatAttribute("User.Title.CasedID")
	if ok := want == got; !ok {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()

	err := NewErrorf(ctx, http.StatusBadRequest, "%d ducks in a row", 4331)
	t.Log(err)

	if errStr := err.Error(); errStr != "Bad Request: 4331 ducks in a row" {
		t.Errorf("invalid error string: %s", errStr)
	}

	wantStr := "4,331 ducks in a row"
	if errMsg := err.LocalizedMessage.Message; errMsg != wantStr {
		t.Errorf("invalid error message: %s", errMsg)
	}

	b, msgErr := json.Marshal(err)
	if msgErr != nil {
		t.Fatal(msgErr)
	}
	t.Log(string(b))

	requestID := xid.New()
	ctx = hlog.CtxWithID(ctx, requestID)
	err = NewErrorf(ctx, http.StatusNotFound, "User not found.")
	t.Log(err)

	if err.RequestInfo.RequestID != requestID.String() {
		t.Errorf("invalid request id: %+v", err)
	}

	err.WithFieldViolation("type", "Goose is not a duck.")
	t.Log(err)

	b, msgErr = json.Marshal(err)
	if msgErr != nil {
		t.Fatal(msgErr)
	}
	t.Log(string(b))

}
