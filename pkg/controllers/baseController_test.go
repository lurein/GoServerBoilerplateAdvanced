package controllers

import (
	"os"
	"testing"

	"whimsy/pkg/migrate"
	"whimsy/pkg/testutils"

	"gorm.io/gorm"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	db = testutils.ConnectDb("controllers")
	if err := migrate.Migrate(db); err != nil {
		panic(err)
	}
	exitVal := m.Run()
	testutils.ResetDb(db)
	os.Exit(exitVal)
}
