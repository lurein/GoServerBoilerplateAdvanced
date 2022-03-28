package models

import (
	"log"
	"os"
	"testing"

	"whimsy/pkg/testutils"

	"gorm.io/gorm"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	db = testutils.ConnectDb("models")

	err := db.AutoMigrate(
		// add in all models here, e.g.
		// &User{},
	)
	if err != nil {
		log.Fatal(err)
	}

	exitVal := m.Run()
	testutils.ResetDb(db)
	os.Exit(exitVal)
}
