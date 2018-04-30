package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	httpHandlers "github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/public-things-api/things"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

func main() {
	app := cli.App("public-things-api-neo4j", "A public RESTful API for accessing Things in neo4j")
	neoURL := app.String(cli.StringOpt{
		Name:   "neo-url",
		Value:  "http://localhost:7474/db/data",
		Desc:   "neo4j endpoint URL",
		EnvVar: "NEO_URL"})
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	graphiteTCPAddress := app.String(cli.StringOpt{
		Name:   "graphiteTCPAddress",
		Value:  "",
		Desc:   "Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)",
		EnvVar: "GRAPHITE_ADDRESS",
	})
	graphitePrefix := app.String(cli.StringOpt{
		Name:   "graphitePrefix",
		Value:  "",
		Desc:   "Prefix to use. Should start with content, include the environment, and the host name. e.g. coco.pre-prod.public-things-api.1",
		EnvVar: "GRAPHITE_PREFIX",
	})
	logMetrics := app.Bool(cli.BoolOpt{
		Name:   "logMetrics",
		Value:  false,
		Desc:   "Whether to log metrics. Set to true if running locally and you want metrics output",
		EnvVar: "LOG_METRICS",
	})
	env := app.String(cli.StringOpt{
		Name:  "env",
		Value: "local",
		Desc:  "environment this app is running in",
	})
	cacheDuration := app.String(cli.StringOpt{
		Name:   "cache-duration",
		Value:  "30s",
		Desc:   "Duration Get requests should be cached for. e.g. 2h45m would set the max-age value to '7440' seconds",
		EnvVar: "CACHE_DURATION",
	})
	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "info",
		Desc:   "Log level of the app",
		EnvVar: "LOG_LEVEL",
	})
	app.Action = func() {
		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)
		log.Infof("public-things-api will listen on port: %s, connecting to: %s", *port, *neoURL)
		driver := driverForNeo4j(*neoURL, *env)
		healthService := &things.HealthService{ThingsDriver:driver}
		runServer(*port, *cacheDuration, healthService, driver)
	}

	log.SetFormatter(&log.JSONFormatter{})
	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithField("LOG_LEVEL", *logLevel).Warn("Cannot parse log level, setting it to INFO.")
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)
	log.WithFields(log.Fields{
		"CACHE_DURATION": *cacheDuration,
		"NEO_URL":        *neoURL,
		"LOG_LEVEL":      *logLevel,
	}).Info("Starting app with arguments")

	log.Infof("Application started with args %s", os.Args)
	app.Run(os.Args)
}

func runServer(port string, cacheDuration string, healthService *things.HealthService, driver things.Driver) {
	var cacheControlHeader string

	if duration, durationErr := time.ParseDuration(cacheDuration); durationErr != nil {
		log.Fatalf("Failed to parse cache duration string, %v", durationErr)
	} else {
		cacheControlHeader = fmt.Sprintf("max-age=%s, public", strconv.FormatFloat(duration.Seconds(), 'f', 0, 64))
	}

	// The following endpoints should not be monitored or logged (varnish calls one of these every second, depending on config)
	// The top one of these build info endpoints feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently same as ping, the content of build-info needs more definition
	// Healthchecks and standards first
	http.HandleFunc(status.PingPath, status.PingHandler)
	http.HandleFunc(status.PingPathDW, status.PingHandler)
	http.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	http.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	http.Handle("/", router(healthService, driver, cacheControlHeader))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
}

func driverForNeo4j(neoURL string, env string) things.Driver {
	conf := neoutils.ConnectionConfig{
		BatchSize:     1024,
		Transactional: false,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
			},
			Timeout: 1 * time.Minute,
		},
		BackgroundConnect: true,
	}
	db, err := neoutils.Connect(neoURL, &conf)
	if err != nil {
		log.Fatalf("Error connecting to neo4j %s", err)
	}
	driver := things.NewCypherDriver(db, env)
	return driver
}

func router(healthService *things.HealthService, driver things.Driver, cacheControlHeader string) http.Handler {
	servicesRouter := mux.NewRouter()

	servicesRouter.Path(status.GTGPath).Handler(handlers.MethodHandler{"GET": http.HandlerFunc(status.NewGoodToGoHandler(healthService.GTG))})
	servicesRouter.Path("/__health").Handler(handlers.MethodHandler{"GET": http.HandlerFunc(healthService.Health())})

	// Then API specific ones:
	thingsHandler := &things.RequestHandler{ThingsDriver:driver, CacheControllerHeader:cacheControlHeader}
	servicesRouter.HandleFunc("/things/{uuid}", thingsHandler.GetThing).Methods("GET")
	servicesRouter.HandleFunc("/things", thingsHandler.GetThings).Methods("GET")
	servicesRouter.HandleFunc("/things/{uuid}", thingsHandler.MethodNotAllowedHandler)

	var monitoringRouter http.Handler = servicesRouter
	monitoringRouter = httpHandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httpHandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)
	return monitoringRouter
}
