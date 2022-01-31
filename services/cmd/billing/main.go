package main

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/billing/infrastructure/integrationevent"
	"arch-homework/pkg/billing/infrastructure/postgres"
	serverhttp "arch-homework/pkg/billing/infrastructure/transport/http"
	"arch-homework/pkg/common/app/streams"
	commonintegrationevent "arch-homework/pkg/common/infrastructure/integrationevent"
	"arch-homework/pkg/common/infrastructure/metrics"
	commonpostgres "arch-homework/pkg/common/infrastructure/postgres"
	infrastreams "arch-homework/pkg/common/infrastructure/streams"
	"arch-homework/pkg/common/jwtauth"

	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	ReadTimeout  = time.Minute
	WriteTimeout = time.Minute
)

const serviceName = "billing"

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Info("service started")

	cfg, err := parseEnv()
	if err != nil {
		logger.Fatal(err)
	}

	connector, err := initDBConnector(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer connector.Close()

	rmqEnv, err := initRabbitMQEnv(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	metricsHandler, err := metrics.NewPrometheusMetricsHandler(metrics.NewDefaultEndpointLabelCollector())
	if err != nil {
		logger.Fatal(err)
	}

	server := startServer(cfg, connector, rmqEnv, logger, metricsHandler)

	waitForKillSignal(logger)
	if err := server.Shutdown(context.Background()); err != nil {
		logger.WithError(err).Fatal("http server shutdown failed")
	}
}

func initDBConnector(cfg *config) (commonpostgres.Connector, error) {
	connector := commonpostgres.NewConnector()
	err := connector.Open(commonpostgres.DSN{
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open database")
	}
	return connector, err
}

func initRabbitMQEnv(cfg *config) (streams.Environment, error) {
	return infrastreams.NewEnvironment(serviceName,
		streams.Config{
			Host:     cfg.RMQHost,
			Port:     cfg.RMQPort,
			User:     cfg.RMQUser,
			Password: cfg.RMQPassword,
		})
}

func waitForKillSignal(logger *logrus.Logger) {
	sysKillSignal := make(chan os.Signal, 1)
	signal.Notify(sysKillSignal, os.Interrupt, syscall.SIGTERM)
	logger.Infof("got system signal '%s'", <-sysKillSignal)
}

func startServer(
	cfg *config,
	connector commonpostgres.Connector,
	rmqEnv streams.Environment,
	logger *logrus.Logger,
	metricsHandler metrics.PrometheusMetricsHandler,
) *http.Server {
	httpAddress := ":" + cfg.ServicePort
	if err := connector.WaitUntilReady(); err != nil {
		logger.Fatal(err)
	}

	trUnitFactory := postgres.NewTransactionalUnitFactory(connector.Client())
	eventHandler := app.NewEventHandler(trUnitFactory, integrationevent.NewEventParser())

	if err := commonintegrationevent.StartEventConsumer(rmqEnv, eventHandler, logger); err != nil {
		logger.Fatal(err)
	}

	tokenParser := jwtauth.NewTokenParser(cfg.JWTSecret)
	_ = tokenParser

	billingQueryService := app.NewBillingQueryService(postgres.NewUserAccountEventRepository(connector.Client()))
	billingService := app.NewBillingService(trUnitFactory)
	billingServer := serverhttp.NewServer(billingService, billingQueryService, tokenParser, logger)

	router := mux.NewRouter()
	router.HandleFunc("/health", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/ready", handleReady(connector)).Methods(http.MethodGet)
	router.PathPrefix(serverhttp.PathPrefix).Handler(billingServer.MakeHandler())
	router.PathPrefix(serverhttp.PathPrefixInternal).Handler(billingServer.MakeInternalHandler())

	metricsHandler.AddMetricsHandler(router, "/metrics")
	metricsHandler.AddCommonMetricsMiddleware(router)

	server := &http.Server{
		Handler:      router,
		Addr:         httpAddress,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
	}

	go func() {
		logger.Fatal(server.ListenAndServe())
	}()

	return server
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, "{\"status\": \"OK\"}")
}

func handleReady(connector commonpostgres.Connector) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if connector.Ready() {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, http.StatusText(http.StatusOK))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}
