package things

import (
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"fmt"
	"github.com/Financial-Times/service-status-go/gtg"
	"net/http"
)

type HealthService struct {}

func (h *HealthService) Health() func(w http.ResponseWriter, r *http.Request) {
	checks := []fthealth.Check{h.HealthCheck()}
	hc := fthealth.HealthCheck{
		SystemCode: "public-things-api",
		Name: "PublicThingsApi",
		Description: "Checks for accessing neo4j",
		Checks: checks,
	}
	return fthealth.Handler(hc)
}

func (h *HealthService) HealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Unable to respond to Public Things api requests",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/public-things-api",
		Severity:         1,
		TechnicalSummary: `Cannot connect to Neo4j. If this check fails, check that Neo4j instance is up and running. You can find the neoUrl as a parameter in hieradata for this service.`,
		Checker:          h.Checker,
	}
}

func (h *HealthService) Checker() (string, error) {
	err := ThingsDriver.checkConnectivity()
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