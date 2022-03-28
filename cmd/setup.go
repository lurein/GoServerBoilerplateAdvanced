package cmd

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
	"whimsy/pkg/controllers"
	"whimsy/pkg/migrate"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newContext() (context.Context, func()) {
	ctx, cancel := newOSSignalContext(context.Background())
	l := log.With().Logger()
	ctx = l.WithContext(ctx)

	return ctx, func() {
		cancel()
	}
}

// newOSSignalContext tries to gracefully handle OS closure.
func newOSSignalContext(ctx context.Context) (context.Context, func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, func() {
		signal.Stop(c)
		cancel()
	}
}

func setupRouter(
	ctx context.Context,
	db *gorm.DB,
	publicKey publicKeyStr,
) *mux.Router {
	router := mux.NewRouter()

	logger := zerolog.Ctx(ctx).With().Caller().Logger() // Add filename and number

	// Setup logging
	router.Use(hlog.NewHandler(logger))
	router.Use(hlog.RequestHandler("request"))
	router.Use(hlog.UserAgentHandler("user_agent"))
	router.Use(hlog.RequestIDHandler("request_id", "X-WHIMSY-REQUEST-ID"))
	router.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		if p := r.URL.Path; strings.HasSuffix(p, "health_check") || strings.HasSuffix(p, "healthCheck") {
			return // ignore
		}
		hlog.FromRequest(r).Info().Int("status_code", status).Int("size", size).Dur("duration", duration).Msg("http request")
	}))

	// Default Routes
	router.NotFoundHandler = http.HandlerFunc(controllers.NotFoundHandler)
	router.HandleFunc("/", controllers.Welcome).Methods("GET")
	router.HandleFunc("/health_check", controllers.HealthCheck).Methods("GET")
	router.HandleFunc("/pk", func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, string(publicKey)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return router

}

func setupDB(ctx context.Context) (*sql.DB, func(), error) {
	if viper.GetString("pg.host") != "" {
		psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
			"password='%s' dbname=%s sslmode=disable", viper.GetString("pg.host"), viper.GetString("pg.port"), viper.GetString("pg.user"), viper.GetString("pg.password"), viper.GetString("pg.dbName"))

		db, err := sql.Open("pgx", psqlInfo)
		if err != nil {
			return nil, nil, err
		}
		return db, func() {
			logger := zerolog.Ctx(ctx)
			if err := db.Close(); err != nil {
				logger.Err(err).Msg("failed to close DB connection")
			}
		}, nil
	}

	// Setup RDS
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		viper.GetString("rds.host"),
		viper.GetString("rds.port"),
		viper.GetString("rds.user"),
		"rds-auth-token-placeholder",
		viper.GetString("rds.dbName"),
	)

	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}

	db := stdlib.OpenDB(*connConfig, stdlib.OptionBeforeConnect(
		func(ctx context.Context, connConfig *pgx.ConnConfig) error {
			sess := session.Must(session.NewSession())

			authToken, err := rdsutils.BuildAuthToken(
				viper.GetString("rds.host")+":"+viper.GetString("rds.port"), // Database Endpoint (With Port)
				viper.GetString("rds.region"),                               // AWS Region
				viper.GetString("rds.user"),                                 // Database Account
				sess.Config.Credentials,
			)
			if err != nil {
				return fmt.Errorf("failed to create authentication token: %w", err)
			}

			connConfig.Password = authToken
			return nil
		},
	))

	return db, func() {
		logger := zerolog.Ctx(ctx)
		if err := db.Close(); err != nil {
			logger.Err(err).Msg("failed to close DB connection")
		}
	}, nil
}

func setupGorm(ctx context.Context, db *sql.DB) (*gorm.DB, error) {
	l := zerolog.Ctx(ctx).With().Logger() // Copy logger
	newLogger := logger.New(
		&l,
		logger.Config{
			SlowThreshold:             2 * time.Second, // Slow SQL threshold
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
		},
	)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	if err := migrate.Migrate(gdb); err != nil {
		return nil, err
	}
	return gdb, nil
}

func setupPrivateKey() (*rsa.PrivateKey, error) {
	var (
		pkeyData []byte
		err      error
	)
	privateKeyPath := viper.GetString("enc.privateKeyPath")
	privateKeyStr := viper.GetString("enc.privateKeyStr")
	if privateKeyPath != "" {
		pkeyData, err = ioutil.ReadFile(privateKeyPath)
		if err != nil {
			return nil, err
		}
	} else if privateKeyStr != "" {
		pkeyData = []byte(privateKeyStr)
	}

	if len(pkeyData) == 0 {
		return nil, fmt.Errorf("private key must be specified as either a path or a string")
	}
	block, _ := pem.Decode([]byte(pkeyData))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		privateKey = k.(*rsa.PrivateKey)
	}
	return privateKey, nil
}

type publicKeyStr string

func setupPublicKey(privateKey *rsa.PrivateKey) (publicKeyStr, error) {
	pubKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}
	publicKeyBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: pubKey}
	var wr bytes.Buffer
	if err := pem.Encode(&wr, publicKeyBlock); err != nil {
		return "", err
	}
	return publicKeyStr(wr.String()), nil
}