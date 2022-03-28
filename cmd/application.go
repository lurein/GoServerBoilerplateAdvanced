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

	root.PersistentFlags().String("logLevel", "debug", "Log level -- trace, debug, info, warn. error")
	bindEnv("logLevel", "LOGLEVEL")

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
