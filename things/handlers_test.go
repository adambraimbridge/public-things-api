package things

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	canonicalUUID string = "00000000-0000-002a-0000-00000000002a"
	alternateUUID string = "00000000-0000-002a-0000-00000000002b"
)

var testConcept = Concept{ID: canonicalUUID, APIURL: canonicalUUID, Types: []string{}}
var testRelationships = []string{"testBroader", "testNarrower"}

func TestGetHandlerSuccess(t *testing.T) {
	expectedBody := `{"id":"` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}`

	d := new(mockedDriver)
	d.On("read", canonicalUUID, []string(nil)).Return(testConcept, true, nil)

	req := newThingsHTTPRequest(t, canonicalUUID, nil)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, expectedBody, rec.Body.String())
	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
}

func TestGetHandlerSuccessWithRelationships(t *testing.T) {
	expectedBody := `{"id":"` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}`

	d := new(mockedDriver)
	d.On("read", canonicalUUID, testRelationships).Return(testConcept, true, nil)

	req := newThingsHTTPRequest(t, canonicalUUID, testRelationships)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, expectedBody, rec.Body.String())
	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
}

func TestGetHandlerNotFound(t *testing.T) {
	expectedBody := message("No thing found with uuid " + canonicalUUID + ".")

	d := new(mockedDriver)
	d.On("read", canonicalUUID, []string(nil)).Return(Concept{}, false, nil)

	req := newThingsHTTPRequest(t, canonicalUUID, nil)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.JSONEq(t, expectedBody, rec.Body.String())
	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
}

func TestGetHandlerReadError(t *testing.T) {
	expectedBody := message("Error getting thing with uuid " + canonicalUUID + ", err=TEST failing to READ")

	d := new(mockedDriver)
	d.On("read", canonicalUUID, []string(nil)).Return(Concept{}, false, errors.New("TEST failing to READ"))

	req := newThingsHTTPRequest(t, canonicalUUID, nil)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, expectedBody, rec.Body.String())
	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
}

func TestGetHandlerRedirect(t *testing.T) {
	d := new(mockedDriver)
	d.On("read", alternateUUID, []string(nil)).Return(testConcept, true, nil)

	req := newThingsHTTPRequest(t, alternateUUID, nil)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "/things/"+canonicalUUID, rec.HeaderMap.Get("Location"))
}

func TestGetHandlerRedirectWithRelationships(t *testing.T) {
	d := new(mockedDriver)
	d.On("read", alternateUUID, testRelationships).Return(testConcept, true, nil)

	req := newThingsHTTPRequest(t, alternateUUID, testRelationships)

	ThingsDriver = d
	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/things/{uuid}", GetThings).Methods("GET")
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "/things/"+canonicalUUID+"?showRelationship=testBroader&showRelationship=testNarrower", rec.HeaderMap.Get("Location"))
}

func newThingsHTTPRequest(t *testing.T, uuid string, relationships []string) *http.Request {
	rUrl := "/things/" + uuid
	if len(relationships) > 0 {
		rUrl += "?"
		v := url.Values{}
		for _, r := range relationships {
			v.Add("showRelationship", r)
		}
		rUrl += v.Encode()
	}

	req, err := http.NewRequest("GET", rUrl, nil)
	require.NoError(t, err)

	return req
}

func message(errMsg string) string {
	return fmt.Sprintf("{\"message\": \"%s\"}\n", errMsg)
}

type mockedDriver struct {
	mock.Mock
}

func (m *mockedDriver) read(id string, relationships []string) (thing Concept, found bool, err error) {
	args := m.Called(id, relationships)
	return args.Get(0).(Concept), args.Bool(1), args.Error(2)
}

func (m *mockedDriver) checkConnectivity() error {
	args := m.Called()
	return args.Error(0)
}
