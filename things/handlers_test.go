package things

import (
	"bytes"
	"fmt"
	"github.com/Financial-Times/go-logger"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	canonicalUUID       = "00000000-0000-002a-0000-00000000002a"
	secondCanonicalUUID = "00000000-0000-002a-0000-00000000002d"
	thirdCanonicalUUID  = "00000000-0000-002a-0000-00000000002f"
	alternateUUID       = "00000000-0000-002a-0000-00000000002b"
	invalidUUID         = "00000000-0000-002a-0000"
)

var testConcept = Thing{ID: canonicalUUID, APIURL: canonicalUUID, Types: []string{}}
var testSecondConcept = Thing{ID: secondCanonicalUUID, APIURL: secondCanonicalUUID, Types: []string{}}
var testThirdConcept = Thing{ID: thirdCanonicalUUID, APIURL: thirdCanonicalUUID, Types: []string{}}
var testRelationships = []string{"testBroader", "testNarrower"}

type mockHTTPClient struct {
	resp       string
	statusCode int
	err        error
}

type testCase struct {
	name         string
	url          string
	clientCode   int
	clientBody   string
	clientError  error
	expectedCode int
	expectedBody string
}

func (mhc *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(mhc.resp)))
	return &http.Response{Body: cb, StatusCode: mhc.statusCode}, mhc.err
}

func TestHandlers(t *testing.T) {
	logger.InitLogger("test service", "debug")
	var mockClient mockHTTPClient
	router := mux.NewRouter()

	getHandlerSuccess := testCase{
		"GetThing - Basic successful request",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebb",
		200,
		getConmpleteThingAsConcept,
		nil,
		200,
		transformedCompleteThing,
	}

	testCases := []testCase{
		getHandlerSuccess,
	}
	for _, test := range testCases {
		mockClient.resp = test.clientBody
		mockClient.statusCode = test.clientCode
		mockClient.err = test.clientError

		handler := NewHandler(&mockClient, "localhost:8080")
		handler.RegisterHandlers(router)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", test.url, nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, test.expectedCode, rr.Code, test.name+" failed: status codes do not match!")
		if rr.Code == 200 {
			fmt.Print(rr.Body.String())
			assert.Equal(t, transformBody(test.expectedBody), rr.Body.String(), test.name+" failed: status body does not match!")
			continue
		}
		assert.Equal(t, test.expectedBody, rr.Body.String(), test.name+" failed: status body does not match!")
	}
}

func transformBody(testBody string) string {
	stripNewLines := strings.Replace(testBody, "\n", "", -1)
	stripTabs := strings.Replace(stripNewLines, "\t", "", -1)
	return stripTabs + "\n"
}

var getConmpleteThingAsConcept = `{
	"id": "http://api.ft.com/things/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"apiUrl": "http://api.ft.com/concepts/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"type": "http://www.ft.com/ontology/product/Brand",
	"prefLabel": "Brussels blog",
	"descriptionXML": "This blog covers everything",
	"imageURL": "http://im.ft-static.com/content/images/2f1be452-02f3-11e6-99cb-83242733f755.png",
	"account": [
		{
			"type": "http://www.ft.com/ontology/twitterHandle",
			"value": "@ftbrussels"
		}
	],
	"alternativeLabels": [
		{
			"type": "http://www.ft.com/ontology/Alias",
			"value": "Brussels Blog"
		}
	],
	"strapline": "Archived",
	"broaderConcepts": [
		{
			"concept": {
				"id": "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
				"apiUrl": "http://api.ft.com/concepts/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
				"type": "http://www.ft.com/ontology/product/Brand",
				"prefLabel": "Financial Times",
				"descriptionXML": "The Financial Times",
				"imageURL": "http://aboutus.ft.com/files/2010/11/ft-logo.gif",
				"alternativeLabels": [
					{
						"type": "http://www.ft.com/ontology/Alias",
						"value": "Financial Times"
					}
				],
				"strapline": "Make the right connections"
			},
			"predicate": "http://www.ft.com/ontology/subBrand"
		}
	]
}`

var transformedCompleteThing = `{
	"id":"http://api.ft.com/things/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"apiUrl":"http://api.ft.com/concepts/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"prefLabel":"Brussels blog",
	"types":[
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/concept/Concept",
		"http://www.ft.com/ontology/classification/Classification",
		"http://www.ft.com/ontology/product/Brand"
	],
	"directType":"http://www.ft.com/ontology/product/Brand",
	"aliases":[
		"Brussels Blog"
	],
	"descriptionXML":"This blog covers everything",
	"_imageUrl":"http://im.ft-static.com/content/images/2f1be452-02f3-11e6-99cb-83242733f755.png",
	"twitterHandle":"@ftbrussels"
}`

// func TestGetHandlerSuccess(t *testing.T) {
// 	expectedBody := `{"id":"` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}`
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(testConcept, true, nil)
//
// 	req := newThingHTTPRequest(t, canonicalUUID, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
// func TestGetThingsHandlerSuccess(t *testing.T) {
// 	expectedBody := `{"things": {
// 						"` + canonicalUUID + `": {"id": "` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]},
// 						"` + secondCanonicalUUID + `": {"id": "` + secondCanonicalUUID + `", "apiUrl":"` + secondCanonicalUUID + `", "types":[]},
// 						"` + thirdCanonicalUUID + `": {"id": "` + thirdCanonicalUUID + `", "apiUrl":"` + thirdCanonicalUUID + `", "types":[]}
// 						}}`
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(testConcept, true, nil)
// 	d.On("read", secondCanonicalUUID, []string(nil)).Return(testSecondConcept, true, nil)
// 	d.On("read", thirdCanonicalUUID, []string(nil)).Return(testThirdConcept, true, nil)
//
// 	req := newThingsHTTPRequest(t, []string{canonicalUUID, secondCanonicalUUID, thirdCanonicalUUID}, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
//
// func TestGetHandlerSuccessWithRelationships(t *testing.T) {
// 	expectedBody := `{"id":"` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}`
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, testRelationships).Return(testConcept, true, nil)
//
// 	req := newThingHTTPRequest(t, canonicalUUID, testRelationships)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
//
// func TestGetHandlerNotFound(t *testing.T) {
// 	expectedBody := message("No thing found with uuid " + canonicalUUID + ".")
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(Thing{}, false, nil)
//
// 	req := newThingHTTPRequest(t, canonicalUUID, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusNotFound, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
//
// func TestGetThingsHandlerNotFound(t *testing.T) {
// 	expectedBody := `{"things":{}}`
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(Thing{}, false, nil)
// 	d.On("read", secondCanonicalUUID, []string(nil)).Return(Concept{}, false, nil)
// 	d.On("read", thirdCanonicalUUID, []string(nil)).Return(Concept{}, false, nil)
//
// 	req := newThingsHTTPRequest(t, []string{canonicalUUID, secondCanonicalUUID, thirdCanonicalUUID}, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
// func TestGetThingsHandlerPartialSuccess(t *testing.T) {
// 	expectedBody := `{"things": {
// 						"` + canonicalUUID + `": {"id": "` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]}}}`
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(testConcept, true, nil)
// 	d.On("read", secondCanonicalUUID, []string(nil)).Return(Concept{}, false, nil)
//
// 	req := newThingsHTTPRequest(t, []string{canonicalUUID, secondCanonicalUUID}, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
//
// func TestGetHandlerReadError(t *testing.T) {
// 	expectedBody := message("Error getting thing with uuid " + canonicalUUID + ", err=TEST failing to READ")
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(Concept{}, false, errors.New("TEST failing to READ"))
//
// 	req := newThingHTTPRequest(t, canonicalUUID, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
// func TestGetThingsHandlerReadError(t *testing.T) {
// 	expectedBody := message("Error getting thing with uuid " + secondCanonicalUUID + ", err=TEST failing to READ")
//
// 	d := new(mockedDriver)
// 	d.On("read", secondCanonicalUUID, []string(nil)).Return(Concept{}, false, errors.New("TEST failing to READ"))
// 	d.On("read", canonicalUUID, []string(nil)).Return(Concept{}, false, nil)
// 	d.On("read", thirdCanonicalUUID, []string(nil)).Return(Concept{}, false, nil)
//
// 	req := newThingsHTTPRequest(t, []string{canonicalUUID, secondCanonicalUUID, thirdCanonicalUUID}, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
// func TestGetThingsHandlerInvalidUUIDError(t *testing.T) {
// 	expectedBody := message("Invalid uuid: " + invalidUUID + ", err: uuid: incorrect UUID length: " + invalidUUID)
//
// 	d := new(mockedDriver)
// 	d.On("read", canonicalUUID, []string(nil)).Return(Concept{}, false, nil)
// 	d.On("read", thirdCanonicalUUID, []string(nil)).Return(Concept{}, false, nil)
//
// 	req := newThingsHTTPRequest(t, []string{canonicalUUID, secondCanonicalUUID, thirdCanonicalUUID, invalidUUID}, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusBadRequest, rec.Code)
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// }
//
// func TestGetHandlerRedirect(t *testing.T) {
// 	d := new(mockedDriver)
// 	d.On("read", alternateUUID, []string(nil)).Return(testConcept, true, nil)
//
// 	req := newThingHTTPRequest(t, alternateUUID, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
// 	assert.Equal(t, "/things/"+canonicalUUID, rec.HeaderMap.Get("Location"))
// }
//
// func TestGetThingsHandlerRedirect(t *testing.T) {
// 	expectedBody := `{"things": {
// 						"` + alternateUUID + `": {"id": "` + canonicalUUID + `", "apiUrl":"` + canonicalUUID + `", "types":[]},
// 						"` + secondCanonicalUUID + `": {"id": "` + secondCanonicalUUID + `", "apiUrl":"` + secondCanonicalUUID + `", "types":[]},
// 						"` + thirdCanonicalUUID + `": {"id": "` + thirdCanonicalUUID + `", "apiUrl":"` + thirdCanonicalUUID + `", "types":[]}
// 						}}`
// 	d := new(mockedDriver)
//
// 	req := newThingsHTTPRequest(t, []string{alternateUUID, secondCanonicalUUID, thirdCanonicalUUID}, nil)
//
// 	d.On("read", alternateUUID, []string(nil)).Return(testConcept, true, nil)
//
// 	d.On("read", canonicalUUID, []string(nil)).Return(testConcept, true, nil)
// 	d.On("read", secondCanonicalUUID, []string(nil)).Return(testSecondConcept, true, nil)
// 	d.On("read", thirdCanonicalUUID, []string(nil)).Return(testThirdConcept, true, nil)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things", handler.GetThings).Methods("GET")
// 	r.ServeHTTP(rec, req)
//
// 	assert.Equal(t, http.StatusOK, rec.Code)
// 	assert.Equal(t, "application/json; charset=UTF-8", rec.HeaderMap.Get("Content-Type"))
// 	assert.JSONEq(t, expectedBody, rec.Body.String())
// }
//
// func TestGetHandlerRedirectWithRelationships(t *testing.T) {
// 	d := new(mockedDriver)
// 	d.On("read", alternateUUID, testRelationships).Return(testConcept, true, nil)
//
// 	req := newThingHTTPRequest(t, alternateUUID, testRelationships)
//
// 	handler := RequestHandler{ThingsDriver: d}
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/things/{uuid}", handler.GetThing).Methods("GET")
// 	r.ServeHTTP(rec, req)
// 	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
// 	assert.Equal(t, "/things/"+canonicalUUID+"?showRelationship=testBroader&showRelationship=testNarrower", rec.HeaderMap.Get("Location"))
// }
//
// func newThingHTTPRequest(t *testing.T, uuid string, relationships []string) *http.Request {
// 	rUrl := "/things/" + uuid
// 	if len(relationships) > 0 {
// 		rUrl += "?"
// 		v := url.Values{}
// 		for _, r := range relationships {
// 			v.Add("showRelationship", r)
// 		}
// 		rUrl += v.Encode()
// 	}
//
// 	req, err := http.NewRequest("GET", rUrl, nil)
// 	require.NoError(t, err)
//
// 	return req
// }
// func newThingsHTTPRequest(t *testing.T, uuids []string, relationships []string) *http.Request {
//
// 	rUrl := "/things?"
// 	params := url.Values{}
// 	for _, uuid := range uuids {
// 		params.Add("uuid", uuid)
// 	}
// 	if len(relationships) > 0 {
// 		for _, r := range relationships {
// 			params.Add("showRelationship", r)
// 		}
// 	}
//
// 	rUrl += params.Encode()
//
// 	req, err := http.NewRequest("GET", rUrl, nil)
// 	require.NoError(t, err)
//
// 	return req
// }
//
// func message(errMsg string) string {
// 	return fmt.Sprintf("{\"message\": \"%s\"}\n", errMsg)
// }
//
// func TestHappyHealthCheck(t *testing.T) {
// 	d := new(mockedDriver)
// 	d.On("checkConnectivity").Return(nil)
//
// 	hs := &HealthService{ThingsDriver: d}
//
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/__health", hs.Health()).Methods("GET")
//
// 	req, err := http.NewRequest("GET", "/__health", nil)
// 	require.NoError(t, err)
//
// 	r.ServeHTTP(rec, req)
//
// 	assert.Equal(t, http.StatusOK, rec.Code)
//
// 	var result fthealth.HealthResult
// 	err = json.NewDecoder(rec.Body).Decode(&result)
// 	assert.NoError(t, err)
// 	assert.Len(t, result.Checks, 1)
// 	assert.True(t, result.Ok)
// 	assert.True(t, result.Checks[0].Ok)
// }
//
// func TestUnhappyHealthCheck(t *testing.T) {
// 	d := new(mockedDriver)
// 	d.On("checkConnectivity").Return(errors.New("computer says no"))
//
// 	hs := &HealthService{ThingsDriver: d}
//
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/__health", hs.Health()).Methods("GET")
//
// 	req, err := http.NewRequest("GET", "/__health", nil)
// 	require.NoError(t, err)
//
// 	r.ServeHTTP(rec, req)
//
// 	assert.Equal(t, http.StatusOK, rec.Code)
//
// 	var result fthealth.HealthResult
// 	err = json.NewDecoder(rec.Body).Decode(&result)
// 	assert.NoError(t, err)
// 	assert.Len(t, result.Checks, 1)
// 	assert.False(t, result.Ok)
// 	assert.False(t, result.Checks[0].Ok)
// 	assert.Equal(t, "computer says no", result.Checks[0].Output)
// }
//
// func TestHealthCheckTimeout(t *testing.T) {
// 	d := new(mockedDriver)
// 	d.On("checkConnectivity").Return(nil).After(11 * time.Second)
//
// 	hs := &HealthService{ThingsDriver: d}
//
// 	rec := httptest.NewRecorder()
// 	r := mux.NewRouter()
// 	r.HandleFunc("/__health", hs.Health()).Methods("GET")
//
// 	req, err := http.NewRequest("GET", "/__health", nil)
// 	require.NoError(t, err)
//
// 	r.ServeHTTP(rec, req)
//
// 	assert.Equal(t, http.StatusOK, rec.Code)
//
// 	var result fthealth.HealthResult
// 	err = json.NewDecoder(rec.Body).Decode(&result)
// 	assert.NoError(t, err)
// 	assert.Len(t, result.Checks, 1)
// 	assert.False(t, result.Ok)
// 	assert.False(t, result.Checks[0].Ok)
// }
//
// type mockedDriver struct {
// 	mock.Mock
// }
//
// func (m *mockedDriver) read(id string, relationships []string) (thing Concept, found bool, err error) {
// 	args := m.Called(id, relationships)
// 	return args.Get(0).(Concept), args.Bool(1), args.Error(2)
// }
//
// func (m *mockedDriver) checkConnectivity() error {
// 	args := m.Called()
// 	return args.Error(0)
// }
