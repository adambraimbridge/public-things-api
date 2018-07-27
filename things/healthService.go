package things

import (
	"fmt"
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

type HealthService struct {
	ThingsDriver Driver
}

func (h *HealthService) Health() func(w http.ResponseWriter, r *http.Request) {
	checks := []fthealth.Check{h.HealthCheck()}
	hc := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "public-things-api",
			Name:        "PublicThingsApi",
			Description: "Checks for accessing neo4j",
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}

	return fthealth.Handler(hc)
}

func (h *HealthService) HealthCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "neo4j-connectivity-check",
		BusinessImpact:   "Unable to respond to Public Things api requests",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://dewey.in.ft.com/view/system/public-things-api",
		Severity:         1,
		TechnicalSummary: `Cannot connect to Neo4j. If this check fails, check that Neo4j instance is up and running.`,
		Checker:          h.Checker,
	}
}

func (h *HealthService) Checker() (string, error) {
	err := h.ThingsDriver.checkConnectivity()
	if err == nil {
		return "Connectivity to neo4j is ok", err
	}
	return "Error connecting to neo4j", err
}

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *HealthService) GTG() gtg.Status {
	check := func() gtg.Status {
		return gtgCheck(h.Checker)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{check})()
}
