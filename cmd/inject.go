//go:build wireinject
// +build wireinject

package cmd

import (
	"context"
	"github.com/google/wire"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func buildRouter(ctx context.Context) (*mux.Router, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		setupDB,
		setupGorm,
		setupPrivateKey,
		setupPublicKey,

		setupRouter,
	)
	return nil, nil, nil
}

func buildGorm(ctx context.Context) (*gorm.DB, func(), error) {
	wire.Build(
		setupDB,
		setupGorm,
	)
	return nil, nil, nil
}
