package things

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	canonicalUUID string = "00000000-0000-002a-0000-00000000002a"
	alternateUUID string = "00000000-0000-002a-0000-00000000002b"
)

type test struct {
	name         string
	req          *http.Request
	dummyService driver
	statusCode   int
	contentType  string // Contents of the Content-Type header
	body         string
}

func TestGetHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []test{
		{"Success", newRequest("GET", fmt.Sprintf("/things/%s", canonicalUUID), "application/json", nil), dummyService{contentUUID: canonicalUUID}, http.StatusOK, "", `{"id":"` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}`},
		{"NotFound", newRequest("GET", fmt.Sprintf("/things/%s", "99999"), "application/json", nil), dummyService{contentUUID: canonicalUUID}, http.StatusNotFound, "", message("No thing found with uuid 99999.")},
		{"ReadError", newRequest("GET", fmt.Sprintf("/things/%s", canonicalUUID), "application/json", nil), dummyService{contentUUID: canonicalUUID, failRead: true}, http.StatusServiceUnavailable, "", message("Error getting thing with uuid " + canonicalUUID + ", err=TEST failing to READ")}}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		r := mux.NewRouter()
		httpHandlers := HttpHandlers{test.dummyService, "max-age=360, public"}
		//httpHandlers.Ser
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.JSONEq(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestGetHandlerForRedirects(t *testing.T) {
	assert := assert.New(t)
	tests := []test{
		{"Redirect", newRequest("GET", fmt.Sprintf("/things/%s", alternateUUID), "application/json", nil), dummyService{contentUUID: canonicalUUID, alternateUUID: alternateUUID}, http.StatusMovedPermanently, "application/json", ""},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		HttpHandlers{test.dummyService, "special header"}).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal("/things/"+canonicalUUID, rec.HeaderMap.Get("Location"), fmt.Sprintf("%s: Wrong location header", test.name))
	}
}

func newRequest(method, url, contentType string, body []byte) *http.Request {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", contentType)
	return req
}

func message(errMsg string) string {
	return fmt.Sprintf("{\"message\": \"%s\"}\n", errMsg)
}

type dummyService struct {
	contentUUID   string
	alternateUUID string
	failRead      bool
}

func (dS dummyService) read(contentUUID string) (thing, bool, error) {
	if dS.failRead {
		return thing{}, false, errors.New("TEST failing to READ")
	}
	if contentUUID == dS.contentUUID || contentUUID == dS.alternateUUID {
		return thing{ID: canonicalUUID, APIURL: canonicalUUID, Types: []string{}}, true, nil
	}
	return thing{}, false, nil
}

func (dS dummyService) checkConnectivity() error {
	return nil
}
