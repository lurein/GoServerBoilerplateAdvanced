package testutils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


func UuidPtr() *uuid.UUID {
	refUuid, _ := uuid.NewV4()
	return &refUuid
}

func UuidStr() string {
	refUuid, _ := uuid.NewV4()
	return refUuid.String()
}

func ConnectDb(packageName string) *gorm.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password='%s' dbname=%s sslmode=disable", os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_TEST_DB"))

	dbconn, err := sql.Open("pgx", psqlInfo)
	if err != nil {
		panic(err)
	}

	dbname := os.Getenv("PG_TEST_DB") + "_" + packageName
	dbconn.Exec("CREATE DATABASE " + dbname)

	dbconn.Close()

	packagePsqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password='%s' dbname=%s sslmode=disable", os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), dbname)

	dbconn, err = sql.Open("pgx", packagePsqlInfo)
	if err != nil {
		panic(err)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: dbconn,
	}))

	if err != nil {
		panic(err)
	}

	return db
}

func ResetDb(db *gorm.DB) {
	// add in all tables below, e.g.
	// db.Exec("TRUNCATE users CASCADE;")
}

func NewContext(t *testing.T) context.Context {
	ctx := context.Background()
	logger := NewLogger(t)
	return logger.WithContext(ctx)
}

func NewLogger(t *testing.T) zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter(zerolog.ConsoleTestWriter(t)))
}

func CmpJSON(t testing.TB, a, b []byte) {
	transformJSON := cmp.FilterValues(func(x, y []byte) bool {
		return json.Valid(x) && json.Valid(y)
	}, cmp.Transformer("ParseJSON", func(in []byte) (out interface{}) {
		if err := json.Unmarshal(in, &out); err != nil {
			panic(err) // should never occur given previous filter to ensure valid JSON
		}
		return out
	}))
	if diff := cmp.Diff(a, b, transformJSON); diff != "" {
		t.Error(diff)
	}
}

type eqCmpMatcher struct {
	want interface{}
	opts cmp.Options
}

// Creates a gomock.Matcher with go-cmp, use for complex struct matching.
func EqCmpMatcher(y interface{}, opts ...cmp.Option) gomock.Matcher {
	return &eqCmpMatcher{
		want: y,
		opts: opts,
	}
}

func (c eqCmpMatcher) Matches(x interface{}) bool {
	return cmp.Equal(c.want, x, c.opts...)
}
func (c eqCmpMatcher) String() string {
	return fmt.Sprintf("Cmp Matcher: %T", c.want)
}
