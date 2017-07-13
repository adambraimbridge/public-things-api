package things

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"log"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/concepts-rw-neo4j/concepts"
	"github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"github.com/Financial-Times/people-rw-neo4j/people"
)

const (
	//Generate uuids so there's no clash with real data
	FakebookConceptUUID    = "eac853f5-3859-4c08-8540-55e043719400" //organisation - Old concept model
	MetalMickeyConceptUUID = "0483bef8-5797-40b8-9b25-b12e492f63c6" //subject - ie. New concept model
	ContentUUID            = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	NonExistingThingUUID   = "b2860919-4b78-44c6-a665-af9221bdefb5"
	PersonThingUUID        = "75e2f7e9-cb5e-40a5-a074-86d69fe09f69"
)

//Reusable Neo4J connection
var db neoutils.NeoConnection

func init() {
	// We are initialising a lot of constraints on an empty database therefore we need the database to be fit before
	// we run tests so initialising the service will create the constraints first
	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, _ = neoutils.Connect(neoUrl(), conf)
	if db == nil {
		panic("Cannot connect to Neo4J")
	}

}

func neoUrl() string {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7777/db/data"
	}
	return url
}

func TestRetrievePeopleAsThing(t *testing.T) {
	assert := assert.New(t)

	personRW := people.NewCypherPeopleService(db)
	assert.NoError(personRW.Initialise())

	writeJSONToService(t, personRW, fmt.Sprintf("./fixtures/People-%s.json", PersonThingUUID))

	defer cleanDB(t, PersonThingUUID)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(PersonThingUUID)
	assert.NoError(err, "Unexpected error for person as thing %s", PersonThingUUID)
	assert.True(found, "Found no thing for person as thing %s", PersonThingUUID)

	validateThing(t, "John Smith", PersonThingUUID, "http://www.ft.com/ontology/person/Person",
		[]string{
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/person/Person",
		}, thng)
}

func TestRetrieveOrganisationAsThing(t *testing.T) {
	assert := assert.New(t)

	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())

	writeJSONToService(t, organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json")

	defer cleanDB(t, FakebookConceptUUID)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(FakebookConceptUUID)
	assert.NoError(err, "Unexpected error for organisation as thing %s", FakebookConceptUUID)
	assert.True(found, "Found no thing for organisation as thing %s", FakebookConceptUUID)

	validateThing(t, "Fakebook, Inc.", FakebookConceptUUID, "http://www.ft.com/ontology/company/PublicCompany",
		[]string{
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/organisation/Organisation",
			"http://www.ft.com/ontology/company/Company",
			"http://www.ft.com/ontology/company/PublicCompany",
		}, thng)
}

func TestRetrieveConceptNewModelAsThing(t *testing.T) {
	conceptsDriver := concepts.NewConceptService(db)
	conceptsDriver.Initialise()

	writeJSONToService(t, conceptsDriver, fmt.Sprintf("./fixtures/Concept-MetalMickey-%s.json", MetalMickeyConceptUUID))

	defer cleanDB(t, MetalMickeyConceptUUID)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(MetalMickeyConceptUUID)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", MetalMickeyConceptUUID)
	assert.True(t, found, "Found no thing for concept as thing %s", MetalMickeyConceptUUID)

	validateThing(t, "Metal Mickey", MetalMickeyConceptUUID, "http://www.ft.com/ontology/Subject",
		[]string{
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/classification/Classification",
			"http://www.ft.com/ontology/Subject",
		}, thng)

}

//TODO - this is temporary, we WILL want to retrieve Content once we have more info about it available
func TestCannotRetrieveContentAsThing(t *testing.T) {
	assert := assert.New(t)

	contentRW := content.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(t, contentRW, "./fixtures/Content-Bitcoin-3fc9fe3e-af8c-4f7f-961a-e5065392bb31.json")

	defer cleanDB(t, NonExistingThingUUID, ContentUUID)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(thing{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func TestRetrieveNoThingsWhenThereAreNonePresent(t *testing.T) {
	assert := assert.New(t)
	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(thing{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func cleanDB(t *testing.T, uuids ...string) {
	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf(`
			MATCH (a:Thing {uuid: "%s"})
			OPTIONAL MATCH (a)-[rel]-(i)
			OPTIONAL MATCH (i)-[rel2]-(d)
			DETACH DELETE rel, rel2, d, i, a`, uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(t, err, "Error executing clean up cypher")
}

func writeJSONToService(t *testing.T, service baseftrwapp.Service, pathToJSONFile string) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(t, err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(t, errr)

	errs := service.Write(inst, "TRANS_ID")
	log.Printf("ERR: %v", errs)
	assert.NoError(t, errs)
}

func validateThing(t *testing.T, prefLabel string, UUID string, directType string, types []string, thng thing) {
	assert.EqualValues(t, prefLabel, thng.PrefLabel, "PrefLabel incorrect")
	assert.EqualValues(t, "http://api.ft.com/things/"+UUID, thng.ID, "ID incorrect")
	assert.EqualValues(t, directType, thng.DirectType, "DirectType incorrect")
	assert.EqualValues(t, types, thng.Types, "Types incorrect")
}
