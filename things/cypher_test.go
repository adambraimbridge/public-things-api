package things

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"reflect"
	"sort"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/concepts-rw-neo4j/concepts"
	"github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/jmcvetta/neoism"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stretchr/testify/assert"
)

const (
	//Generate uuids so there's no clash with real data
	FakebookConceptUUID  = "eac853f5-3859-4c08-8540-55e043719400" //organisation - Old concept model
	ContentUUID          = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	NonExistingThingUUID = "b2860919-4b78-44c6-a665-af9221bdefb5"
	TopicOnyxPike        = "9a07c16f-def0-457d-a04a-57ba68ba1e00"
	TopicOnyxPikeRelated = "ec20c787-8289-4cef-aee8-4d39e9563dc5"
	TopicOnyxPikeBroader = "ba42b8d0-844f-4f2a-856c-5cbd863bf6bd"
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
		url = "http://localhost:7474/db/data"
	}
	return url
}

func TestRetrieveOrganisationAsThing(t *testing.T) {
	defer cleanDB(t, FakebookConceptUUID)

	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(t, organisationRW.Initialise())

	types := []string{"Thing", "Concept", "Organisation", "Company", "PublicCompany"}
	typesUris := mapper.TypeURIs(types)

	expected := Concept{APIURL: mapper.APIURL(FakebookConceptUUID, types, "Prod"), PrefLabel: "Fakebook, Inc.", ID: mapper.IDURL(FakebookConceptUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1]}

	writeJSONToService(t, organisationRW, fmt.Sprintf("./fixtures/Organisation-Fakebook-%v.json", FakebookConceptUUID))

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(FakebookConceptUUID, nil)
	assert.NoError(t, err, "Unexpected error for organisation as thing %s", FakebookConceptUUID)
	assert.True(t, found, "Found no Concept for organisation as Concept %s", FakebookConceptUUID)

	readAndCompare(t, expected, thng, "Organisation successful retrieval")
}

func TestRetrieveConceptNewModelAsThing(t *testing.T) {
	defer cleanDB(t, TopicOnyxPikeRelated, TopicOnyxPikeBroader, TopicOnyxPike)

	conceptsDriver := concepts.NewConceptService(db)
	assert.NoError(t, conceptsDriver.Initialise())

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expected := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		BroaderConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
		RelatedConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeRelated), APIURL: mapper.APIURL(TopicOnyxPikeRelated, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Related", DirectType: typesUris[len(typesUris)-1], Predicate: skosRelatedURI}}}

	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPikeRelated-%s.json", TopicOnyxPikeRelated))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPikeBroader-%s.json", TopicOnyxPikeBroader))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPike-%s.json", TopicOnyxPike))

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{"broader", "related"}
	thng, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expected, thng, "Retrieve concepts via new concordance model")
}

func readAndCompare(t *testing.T, expected Concept, actual Concept, testName string) {
	sort.Slice(expected.Aliases, func(i, j int) bool {
		return expected.Aliases[i] < expected.Aliases[j]
	})

	sort.Slice(actual.Aliases, func(i, j int) bool {
		return actual.Aliases[i] < actual.Aliases[j]
	})

	sort.Slice(expected.Types, func(i, j int) bool {
		return expected.Types[i] < expected.Types[j]
	})

	sort.Slice(actual.Types, func(i, j int) bool {
		return actual.Types[i] < actual.Types[j]
	})

	sort.Slice(actual.BroaderConcepts, func(i, j int) bool {
		return actual.BroaderConcepts[i].ID < actual.BroaderConcepts[j].ID
	})

	sort.Slice(actual.NarrowerConcepts, func(i, j int) bool {
		return actual.NarrowerConcepts[i].ID < actual.NarrowerConcepts[j].ID
	})

	sort.Slice(actual.RelatedConcepts, func(i, j int) bool {
		return actual.RelatedConcepts[i].ID < actual.RelatedConcepts[j].ID
	})

	sort.Slice(expected.BroaderConcepts, func(i, j int) bool {
		return expected.BroaderConcepts[i].ID < expected.BroaderConcepts[j].ID
	})

	sort.Slice(expected.NarrowerConcepts, func(i, j int) bool {
		return expected.NarrowerConcepts[i].ID < expected.NarrowerConcepts[j].ID
	})

	sort.Slice(expected.RelatedConcepts, func(i, j int) bool {
		return expected.RelatedConcepts[i].ID < expected.RelatedConcepts[j].ID
	})

	for _, thing := range actual.NarrowerConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}
	for _, thing := range actual.BroaderConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}
	for _, thing := range actual.RelatedConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}

	for _, thing := range expected.NarrowerConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}
	for _, thing := range expected.BroaderConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}
	for _, thing := range expected.RelatedConcepts {
		sort.Slice(thing.Types, func(i, j int) bool {
			return thing.Types[i] < thing.Types[j]
		})
	}

	assert.True(t, reflect.DeepEqual(expected, actual), fmt.Sprintf("Actual concept differs from expected: \n ExpectedConcept: %v \n Actual: %v", expected, actual))
}

//TODO - this is temporary, we WILL want to retrieve Content once we have more info about it available
func TestCannotRetrieveContentAsThing(t *testing.T) {
	assert := assert.New(t)

	contentRW := content.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(t, contentRW, "./fixtures/Content-Bitcoin-3fc9fe3e-af8c-4f7f-961a-e5065392bb31.json")

	defer cleanDB(t, NonExistingThingUUID, ContentUUID)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID, nil)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(Concept{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func TestRetrieveNoThingsWhenThereAreNonePresent(t *testing.T) {
	assert := assert.New(t)
	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID, nil)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(Concept{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func cleanDB(t *testing.T, uuids ...string) {
	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf(`
			MATCH (a:Thing {uuid: "%s"})
			OPTIONAL MATCH (a)-[ids:IDENTIFIES]-(c)
			OPTIONAL MATCH (related)-[rel]-(d)
			DELETE ids, rel
			DETACH DELETE c, d, a`, uuid)}
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
	assert.NoError(t, errs)
}

func writeJSONToConceptsService(t *testing.T, service concepts.ConceptService, pathToJSONFile string) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(t, err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(t, errr)

	_, errs := service.Write(inst, "TRANS_ID")
	assert.NoError(t, errs)
}

func validateThing(t *testing.T, prefLabel string, UUID string, directType string, types []string, thng Concept) {
	assert.EqualValues(t, prefLabel, thng.PrefLabel, "PrefLabel incorrect")
	assert.EqualValues(t, "http://api.ft.com/Things/"+UUID, thng.ID, "ID incorrect")
	assert.EqualValues(t, directType, thng.DirectType, "DirectType incorrect")
	assert.EqualValues(t, types, thng.Types, "Types incorrect")
}
