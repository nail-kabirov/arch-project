package main

import (
	"arch-homework/pkg/common/app/streams"
	commonintegrationevent "arch-homework/pkg/common/infrastructure/integrationevent"
	"arch-homework/pkg/common/infrastructure/metrics"
	commonpostgres "arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/common/infrastructure/storedevent"
	infrastreams "arch-homework/pkg/common/infrastructure/streams"
	"arch-homework/pkg/common/jwtauth"
	"arch-homework/pkg/lot/app"
	"arch-homework/pkg/lot/infrastructure/billing"
	"arch-homework/pkg/lot/infrastructure/integrationevent"
	"arch-homework/pkg/lot/infrastructure/postgres"
	serverhttp "arch-homework/pkg/lot/infrastructure/transport/http"
	"arch-homework/pkg/lot/infrastructure/user"

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

const serviceName = "lot"

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

	metricsHandler, err := metrics.NewPrometheusMetricsHandler(serverhttp.NewEndpointLabelCollector())
	if err != nil {
		logger.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	server := startServer(ctx, cfg, connector, rmqEnv, logger, metricsHandler)

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
	ctx context.Context,
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
	dbDep := postgres.NewDBDependency(connector.Client())
	eventStore, err := storedevent.NewEventSender(ctx, postgres.NewEventStore(connector.Client()), rmqEnv, logger)
	if err != nil {
		logger.Fatal(err)
	}
	billingClient := billing.NewClient(http.Client{}, cfg.BillingServiceHost)
	userClient := user.NewClient(http.Client{}, cfg.UserServiceHost)

	eventHandler := app.NewEventHandler(dbDep, integrationevent.NewEventParser(), eventStore, billingClient)
	if err := commonintegrationevent.StartEventConsumer(rmqEnv, eventHandler, logger); err != nil {
		logger.Fatal(err)
	}

	lotService := app.NewLotService(dbDep, dbDep, eventStore, billingClient)
	lotQueryService := postgres.NewLotQueryService(connector.Client(), userClient)
	tokenParser := jwtauth.NewTokenParser(cfg.JWTSecret)
	lotServer := serverhttp.NewServer(lotService, lotQueryService, tokenParser, logger)

	app.StartCompletedLotsHandler(ctx, lotService, logger)

	router := mux.NewRouter()
	router.HandleFunc("/health", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/ready", handleReady(connector)).Methods(http.MethodGet)
	router.PathPrefix(serverhttp.PathPrefix).Handler(lotServer.MakeHandler())
	router.PathPrefix(serverhttp.PathPrefixInternal).Handler(lotServer.MakeInternalHandler())

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
