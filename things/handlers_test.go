package things

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/go-logger"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	getThingSuccess := testCase{
		"GetThing - Basic successful request",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebb",
		200,
		getConmpleteThingAsConcept,
		nil,
		200,
		transformedCompleteThing,
	}

	getThingSuccessWithRelationShip := testCase{
		"GetThing - Basic successful request with relationship parameter",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebb?showRelationship=related",
		200,
		getConmpleteThingWithRelationAsConcept,
		nil,
		200,
		transformedCompleteThingWithRelation,
	}

	getThingNotFound := testCase{
		"GetThing - request is not found",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebc",
		404,
		"",
		nil,
		404,
		`{"message":"No thing found with uuid 6773e864-78ab-4051-abc2-f4e9ab423ebc."}`,
	}

	getThingRedirect := testCase{
		"GetThing - redirect",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebc",
		200,
		getConmpleteThingAsConcept,
		nil,
		301,
		``,
	}

	getThingRedirectWithRelationships := testCase{
		"GetThing - redirect with relationships parameter",
		"/things/6773e864-78ab-4051-abc2-f4e9ab423ebc?showRelationship=narrower",
		200,
		getConmpleteThingAsConcept,
		nil,
		301,
		``,
	}

	getThingsWithoutParams := testCase{
		"GetThings - request with no query parameter",
		"/things",
		400,
		"",
		nil,
		400,
		`{"message":"at least one uuid query param should be provided for batch operations"}`,
	}

	getThingsNotFound := testCase{
		"GetThings - request with valid format UUID, but not exist",
		"/things?uuid=6773e864-78ab-4051-abc2-f4e9ab423ebc",
		404,
		"",
		nil,
		200,
		`{"things":{}}`,
	}

	getThingsWithAlternativeUUID := testCase{
		"GetThings - request with alternative uuid, which returns canonical uuid",
		"/things?uuid=6773e864-78ab-4051-abc2-f4e9ab423ebc",
		200,
		getConmpleteThingAsConcept,
		nil,
		200,
		`{"things":{"6773e864-78ab-4051-abc2-f4e9ab423ebc":` + transformedCompleteThing + `}}`,
	}

	testCases := []testCase{
		getThingSuccess,
		getThingSuccessWithRelationShip,
		getThingNotFound,
		getThingRedirect,
		getThingRedirectWithRelationships,
		getThingsWithoutParams,
		getThingsNotFound,
		getThingsWithAlternativeUUID,
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
		assert.Equal(t, "application/json; charset=UTF-8", rr.Header().Get("Content-Type"))
		assert.Equal(t, test.expectedCode, rr.Code, test.name+" failed: status codes do not match!")
		if rr.Code == http.StatusOK {
			fmt.Print(rr.Body.String())
			assert.Equal(t, transformBody(test.expectedBody), rr.Body.String(), test.name+" failed: status body does not match!")
			continue
		}
		assert.Equal(t, test.expectedBody, rr.Body.String(), test.name+" failed: status body does not match!")
	}
}

func TestHappyHealthCheck(t *testing.T) {
	d := new(mockedDriver)
	d.On("checkConnectivity").Return(nil)

	hs := &HealthService{ThingsDriver: d}

	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/__health", hs.Health()).Methods("GET")

	req, err := http.NewRequest("GET", "/__health", nil)
	require.NoError(t, err)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result fthealth.HealthResult
	err = json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Checks, 1)
	assert.True(t, result.Ok)
	assert.True(t, result.Checks[0].Ok)
}

func TestUnhappyHealthCheck(t *testing.T) {
	d := new(mockedDriver)
	d.On("checkConnectivity").Return(errors.New("computer says no"))

	hs := &HealthService{ThingsDriver: d}

	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/__health", hs.Health()).Methods("GET")

	req, err := http.NewRequest("GET", "/__health", nil)
	require.NoError(t, err)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result fthealth.HealthResult
	err = json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Checks, 1)
	assert.False(t, result.Ok)
	assert.False(t, result.Checks[0].Ok)
	assert.Equal(t, "computer says no", result.Checks[0].CheckOutput)
}

func TestHealthCheckTimeout(t *testing.T) {
	d := new(mockedDriver)
	d.On("checkConnectivity").Return(nil).After(11 * time.Second)

	hs := &HealthService{ThingsDriver: d}

	rec := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/__health", hs.Health()).Methods("GET")

	req, err := http.NewRequest("GET", "/__health", nil)
	require.NoError(t, err)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result fthealth.HealthResult
	err = json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Checks, 1)
	assert.False(t, result.Ok)
	assert.False(t, result.Checks[0].Ok)
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
	"strapline": "Archived"
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

var getConmpleteThingWithRelationAsConcept = `{
	"id": "http://api.ft.com/things/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"apiUrl": "http://api.ft.com/concepts/6773e864-78ab-4051-abc2-f4e9ab423ebb",
	"type": "http://www.ft.com/ontology/product/Brand",
	"prefLabel": "Brussels blog",
	"descriptionXML": "This blog covers everything",
	"imageURL": "http://im.ft-static.com/content/images/2f1be452-02f3-11e6-99cb-83242733f755.png",
	"account": [],
	"alternativeLabels": [],
	"relatedConcepts": [
		{
			"concept": {
				"id": "http://api.ft.com/things/50d8fd9f-c4f3-42ae-9467-84a842c3c829",
				"apiUrl": "http://api.ft.com/concepts/50d8fd9f-c4f3-42ae-9467-84a842c3c829",
				"type": "http://www.ft.com/ontology/Topic",
				"prefLabel": "Neglected tropical diseases",
				"alternativeLabels": []
			},
			"predicate": "http://www.ft.com/ontology/related"
		}
	]
}`

var transformedCompleteThingWithRelation = `{
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
	"descriptionXML":"This blog covers everything",
	"_imageUrl":"http://im.ft-static.com/content/images/2f1be452-02f3-11e6-99cb-83242733f755.png",
	"relatedConcepts":[
		{
			"id":"http://api.ft.com/things/50d8fd9f-c4f3-42ae-9467-84a842c3c829",
			"apiUrl":"http://api.ft.com/concepts/50d8fd9f-c4f3-42ae-9467-84a842c3c829",
			"prefLabel":"Neglected tropical diseases",
			"types":[
				"http://www.ft.com/ontology/core/Thing",
				"http://www.ft.com/ontology/concept/Concept",
				"http://www.ft.com/ontology/Topic"
			],
			"directType":"http://www.ft.com/ontology/Topic",
			"predicate":"http://www.ft.com/ontology/related"
		}
	]
}`

func transformBody(testBody string) string {
	stripNewLines := strings.Replace(testBody, "\n", "", -1)
	stripTabs := strings.Replace(stripNewLines, "\t", "", -1)
	return stripTabs + "\n"
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
