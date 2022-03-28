package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var root = &cobra.Command{
	Use:   "whimsy",
	Short: "Whimsy backend",
}

func bindEnv(key, val string) {
	if err := viper.BindEnv(key, val); err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	root.PersistentFlags().String("rds.host", "", "RDS Endpoint")
	bindEnv("rds.host", "RDS_HOST")
	root.PersistentFlags().String("rds.port", "5432", "RDS Endpoint")
	bindEnv("rds.port", "RDS_PORT")
	root.PersistentFlags().String("rds.dbName", "", "RDS DB Name")
	bindEnv("rds.dbName", "RDS_DBNAME")
	root.PersistentFlags().String("rds.region", "", "RDS DB Region")
	bindEnv("rds.region", "RDS_REGION")
	root.PersistentFlags().String("rds.user", "whimsy", "RDS DB User")
	bindEnv("rds.user", "RDS_USER")

	root.PersistentFlags().String("logLevel", "debug", "Log level -- trace, debug, info, warn. error")
	bindEnv("logLevel", "LOGLEVEL")

	root.PersistentFlags().String("pg.host", "", "PG Endpoint")
	root.PersistentFlags().String("pg.port", "5432", "PG Endpoint")
	root.PersistentFlags().String("pg.dbName", "whimsy", "PG DB Name")
	root.PersistentFlags().String("pg.user", "postgres", "PG DB User")
	root.PersistentFlags().String("pg.password", "password", "PG DB Password")

	root.PersistentFlags().String("enc.privateKeyStr", "", "encryption private key as a string, PEM encoded")
	bindEnv("enc.privateKeyStr", "ENC_PRIVATE_KEY_STR")
	root.PersistentFlags().String("enc.privateKeyPath", "", "Path to encryption private key, PEM encoded")
	bindEnv("enc.privateKeyPath", "ENC_PRIVATE_KEY_PATH")

	// server Flags
	root.PersistentFlags().String("http.address", ":5000", "Launch the app, visit localhost:5000/")
	bindEnv("http.address", "HTTP_ADDRESS")
	viper.BindPFlags(serverCmd.PersistentFlags())

	viper.BindPFlags(root.PersistentFlags())
}

func initConfig() {
	switch viper.GetString("logLevel") {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}

func Execute() {
	viper.AutomaticEnv()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
