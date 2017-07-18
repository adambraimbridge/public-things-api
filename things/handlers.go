package things

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/gorilla/mux"
)

var ThingsDriver driver
var CacheControlHeader string

const validUUID = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"

//var maxAge = 24 * time.Hour

func HealthCheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Unable to respond to Public Things api requests",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/public-things-api",
		Severity:         1,
		TechnicalSummary: `Cannot connect to Neo4j. If this check fails, check that Neo4j instance is up and running. You can find the neoUrl as a parameter in hieradata for this service.`,
		Checker:          Checker,
	}
}

func Checker() (string, error) {
	err := ThingsDriver.checkConnectivity()
	if err == nil {
		return "Connectivity to neo4j is ok", err
	}
	return "Error connecting to neo4j", err
}

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

//goodToGo returns a 503 if the healthcheck fails - suitable for use from varnish to check availability of a node
func GoodToGo(writer http.ResponseWriter, req *http.Request) {
	if _, err := Checker(); err != nil {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}

}

// buildInfoHandler - This is a stop gap and will be added to when we can define what we should display here
func BuildInfoHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "build-info")
}

// methodNotAllowedHandler handles 405
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

func GetThings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if uuid == "" {
		http.Error(w, "uuid required", http.StatusBadRequest)
		return
	}
	thng, found, err := ThingsDriver.read(uuid)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := fmt.Sprintf(`{"message":"Error getting thing with uuid %s, err=%s"}`, uuid, err.Error())
		w.Write([]byte(msg))
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf(`{"message":"No thing found with uuid %s."}`, uuid)
		w.Write([]byte(msg))
		return
	}

	//if the request was not made for the canonical, but an alternate uuid: redirect
	if !strings.Contains(thng.ID, uuid) {
		validRegexp := regexp.MustCompile(validUUID)
		canonicalUUID := validRegexp.FindString(thng.ID)
		redirectURL := strings.Replace(r.URL.String(), uuid, canonicalUUID, 1)
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Cache-Control", CacheControlHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(thng); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf(`{"message":"Error parsing thing with uuid %s, err=%s"}`, uuid, err.Error())
		w.Write([]byte(msg))
	}
}
