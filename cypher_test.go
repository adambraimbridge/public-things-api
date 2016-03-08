package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/subjects-rw-neo4j/subjects"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
//Generate uuids so there's no clash with real data
	FakebookConceptUUID = "eac853f5-3859-4c08-8540-55e043719400" //organization
	MetalMickeyConceptUUID = "0483bef8-5797-40b8-9b25-b12e492f63c6" //subject
	NonExistingThingUUID = "b2860919-4b78-44c6-a665-af9221bdefb5"
)

func TestRetrieveOrganizationAsThing(t *testing.T) {
	assert := assert.New(t)
	expectedThing := getExpectedFakebookThing()
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)


	organisationRW := writeOrganisation(assert, db, &batchRunner)

	defer cleanDB(db, t, assert)
	defer deleteOrganisation(organisationRW)

	thingsDriver := newCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(FakebookConceptUUID)
	assert.NoError(err, "Unexpected error for content %s", FakebookConceptUUID)
	assert.True(found, "Found no thing for content %s", FakebookConceptUUID)
	assert.EqualValues(expectedThing, thng, "Didn't get the thing")
}

func TestRetrieveSubjectAsThing(t *testing.T) {
	assert := assert.New(t)
	expectedThing := getExpectedMetalMickeyThing()
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)
	subjectRW := writeSubject(assert, db, &batchRunner)

	defer cleanDB(db, t, assert)
	defer deleteSubject(subjectRW)

	thingsDriver := newCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(MetalMickeyConceptUUID)
	assert.NoError(err, "Unexpected error for content %s", MetalMickeyConceptUUID)
	assert.True(found, "Found no thing for content %s", MetalMickeyConceptUUID)
	assert.EqualValues(expectedThing, thng, "Didn't get the thing")
}

func TestRetrieveNoThingsWhenThereAreNonePresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	defer cleanDB(db, t, assert)

	thingsDriver := newCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(thing{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}


func writeOrganisation(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	organisationRW := organisations.NewCypherOrganisationService(*batchRunner, db)
	assert.NoError(organisationRW.Initialise())
	writeJSONToService(organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json", assert)
	return organisationRW
}

func deleteOrganisation(organisationRW baseftrwapp.Service) {
	organisationRW.Delete(FakebookConceptUUID)
}

func writeSubject(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	subjectsRW := subjects.NewCypherSubjectsService(*batchRunner, db)
	assert.NoError(subjectsRW.Initialise())
	writeJSONToService(subjectsRW, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json", assert)
	return subjectsRW
}

func deleteSubject(subjectsRW baseftrwapp.Service) {
	subjectsRW.Delete(MetalMickeyConceptUUID)
}

func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) *neoism.Database {
	db := getDatabaseConnection(t, assert)
	cleanDB(db, t, assert)
	return db
}

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) *neoism.Database {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db *neoism.Database, t *testing.T, assert *assert.Assertions) {
	uuids := []string{
		FakebookConceptUUID,
		MetalMickeyConceptUUID,
	}

	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf("MATCH (a:Thing{uuid: '%s'}) DETACH DELETE a", uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func getExpectedFakebookThing() thing {
	return thing{
		ID:        "http://api.ft.com/things/eac853f5-3859-4c08-8540-55e043719400",
		APIURL:    "http://api.ft.com/organisations/eac853f5-3859-4c08-8540-55e043719400",
		Types: []string{
			"http://www.ft.com/ontology/organisation/Organisation",
			"http://www.ft.com/ontology/company/PublicCompany",
			"http://www.ft.com/ontology/company/Company",
		},
		DirectType: "http://www.ft.com/ontology/organisation/Organisation",
		PrefLabel: "Fakebook, Inc.",
	}
}


func getExpectedMetalMickeyThing() thing {
	return thing{
		ID:        "http://api.ft.com/things/0483bef8-5797-40b8-9b25-b12e492f63c6",
		APIURL:    "http://api.ft.com/things/0483bef8-5797-40b8-9b25-b12e492f63c6",
		Types: []string{
			"http://www.ft.com/ontology/Subject",
		},
		DirectType:"http://www.ft.com/ontology/Subject",
		PrefLabel: "Metal Mickey",
	}
}
