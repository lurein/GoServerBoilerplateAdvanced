package migrate

import (
	"fmt"
	//"whimsy/pkg/models"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	for i, v := range []interface{}{
		// add in all models below:
		// e.g.
		//&models.User{},
	} {
		if err := db.AutoMigrate(v); err != nil {
			log.Err(err).Int("step", i).Str("type", fmt.Sprintf("%T", v)).Msg("migration failed")
			return err
		}
	}
	return nil
}
