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
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/jmcvetta/neoism"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stretchr/testify/assert"
)

const (
	//Generate uuids so there's no clash with real data
	FakebookConceptUUID            = "eac853f5-3859-4c08-8540-55e043719400" //organisation - Old concept model
	ContentUUID                    = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	NonExistingThingUUID           = "b2860919-4b78-44c6-a665-af9221bdefb5"
	TopicOnyxPike                  = "9a07c16f-def0-457d-a04a-57ba68ba1e00"
	TopicOnyxPikeRelated           = "ec20c787-8289-4cef-aee8-4d39e9563dc5"
	TopicOnyxPikeBroader           = "ba42b8d0-844f-4f2a-856c-5cbd863bf6bd"
	TopicOnyxPikeBroaderTransitive = "a0ec2c50-1174-48f2-b804-d1f346bb7256"
	BrandFTUUID                    = "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
	BrandLexUUID                   = "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"
	BrandLexLiveUUID               = "e363dfb8-f6d9-4f2c-beba-5162b334272b"
)

//Reusable Neo4J connection
var db neoutils.NeoConnection

func init() {
	// We are initialising a lot of constraints on an empty database therefore we need the database to be fit before
	// we run tests so initialising the service will create the constraints first
	logger.InitDefaultLogger("test-service")
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

func TestRetrieveConceptAsThingWithoutRelationships(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expected := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
	}

	thingsDriver := NewCypherDriver(db, "prod")

	thng, found, err := thingsDriver.read(TopicOnyxPike, nil)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expected, thng, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptWithBroader(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedOnyxPike := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		BroaderConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	expectedOnyxPikeBroader := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), PrefLabel: "Onyx Pike Broader", ID: mapper.IDURL(TopicOnyxPikeBroader),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Business & Economy", "Onyx Pike Broader"},
		BroaderConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{broader}
	actualOnyxPike, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedOnyxPike, actualOnyxPike, "Retrieve concepts via new concordance model")

	actualOnyxPikeBroader, found, err := thingsDriver.read(TopicOnyxPikeBroader, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPikeBroader)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPikeBroader)

	readAndCompare(t, expectedOnyxPikeBroader, actualOnyxPikeBroader, "Retrieve concepts via new concordance model")
}

func TestRetrieveBrandWithBroader(t *testing.T) {
	createLexLiveScenario(t)
	defer cleanUpLexLiveScenario(t)

	types := []string{"Thing", "Concept", "Classification", "Brand"}
	typesUris := mapper.TypeURIs(types)

	expectedLexLive := Concept{APIURL: mapper.APIURL(BrandLexLiveUUID, types, "Prod"), PrefLabel: "Lex Live", ID: mapper.IDURL(BrandLexLiveUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Lex Live"},
		BroaderConcepts: []Thing{{ID: mapper.IDURL(BrandLexUUID), APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), Types: typesUris, PrefLabel: "Lex", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	expectedLex := Concept{APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), PrefLabel: "Lex", ID: mapper.IDURL(BrandLexUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"LEX", "Lex"},
		BroaderConcepts: []Thing{{ID: mapper.IDURL(BrandFTUUID), APIURL: mapper.APIURL(BrandFTUUID, types, "Prod"), Types: typesUris, PrefLabel: "Financial Times", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{broader}
	actualLexLive, found, err := thingsDriver.read(BrandLexLiveUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandLexLiveUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandLexLiveUUID)

	readAndCompare(t, expectedLexLive, actualLexLive, "Retrieve concepts via new concordance model")

	actualLex, found, err := thingsDriver.read(BrandLexUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandLexUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandLexUUID)

	readAndCompare(t, expectedLex, actualLex, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptWithBroaderTransitive(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedOnyxPike := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		BroaderConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI},
			{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderTransitiveURI},
		},
	}

	expectedOnyxPikeBroader := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), PrefLabel: "Onyx Pike Broader", ID: mapper.IDURL(TopicOnyxPikeBroader),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Business & Economy", "Onyx Pike Broader"},
		BroaderConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{broaderTransitive}
	actualOnyxPike, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedOnyxPike, actualOnyxPike, "Retrieve concepts via new concordance model")

	actualOnyxPikeBroader, found, err := thingsDriver.read(TopicOnyxPikeBroader, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPikeBroader)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPikeBroader)

	readAndCompare(t, expectedOnyxPikeBroader, actualOnyxPikeBroader, "Retrieve concepts via new concordance model")
}

func TestRetrieveBrandWithBroaderTransitive(t *testing.T) {
	createLexLiveScenario(t)
	defer cleanUpLexLiveScenario(t)

	types := []string{"Thing", "Concept", "Classification", "Brand"}
	typesUris := mapper.TypeURIs(types)

	expectedLexLive := Concept{APIURL: mapper.APIURL(BrandLexLiveUUID, types, "Prod"), PrefLabel: "Lex Live", ID: mapper.IDURL(BrandLexLiveUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Lex Live"},
		BroaderConcepts: []Thing{
			{ID: mapper.IDURL(BrandLexUUID), APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), Types: typesUris, PrefLabel: "Lex", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI},
			{ID: mapper.IDURL(BrandFTUUID), APIURL: mapper.APIURL(BrandFTUUID, types, "Prod"), Types: typesUris, PrefLabel: "Financial Times", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderTransitiveURI},
		},
	}

	expectedLex := Concept{APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), PrefLabel: "Lex", ID: mapper.IDURL(BrandLexUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"LEX", "Lex"},
		BroaderConcepts: []Thing{{ID: mapper.IDURL(BrandFTUUID), APIURL: mapper.APIURL(BrandFTUUID, types, "Prod"), Types: typesUris, PrefLabel: "Financial Times", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{broaderTransitive}
	actualLexLive, found, err := thingsDriver.read(BrandLexLiveUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandLexLiveUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandLexLiveUUID)

	readAndCompare(t, expectedLexLive, actualLexLive, "Retrieve concepts via new concordance model")

	actualLex, found, err := thingsDriver.read(BrandLexUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandLexUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandLexUUID)

	readAndCompare(t, expectedLex, actualLex, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptWithNarrower(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedOnyxPikeBroader := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), PrefLabel: "Onyx Pike Broader", ID: mapper.IDURL(TopicOnyxPikeBroader),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Business & Economy", "Onyx Pike Broader"},
		NarrowerConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPike), APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike", DirectType: typesUris[len(typesUris)-1], Predicate: skosNarrowerURI}},
	}

	expectedOnyxPikeBroaderTransitive := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), PrefLabel: "Onyx Pike Broader Transitive", ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Transitive Business & Economy", "Onyx Pike Broader Transitive"},
		NarrowerConcepts: []Thing{{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosNarrowerURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{narrower}
	actualOnyxPikeBroader, found, err := thingsDriver.read(TopicOnyxPikeBroader, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedOnyxPikeBroader, actualOnyxPikeBroader, "Retrieve concepts via new concordance model")

	actualOnyxPikeBroaderTransitive, found, err := thingsDriver.read(TopicOnyxPikeBroaderTransitive, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPikeBroader)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPikeBroader)

	readAndCompare(t, expectedOnyxPikeBroaderTransitive, actualOnyxPikeBroaderTransitive, "Retrieve concepts via new concordance model")
}

func TestRetrieveBrandWithNarrower(t *testing.T) {
	createLexLiveScenario(t)
	defer cleanUpLexLiveScenario(t)

	types := []string{"Thing", "Concept", "Classification", "Brand"}
	typesUris := mapper.TypeURIs(types)

	expectedFT := Concept{APIURL: mapper.APIURL(BrandFTUUID, types, "Prod"), PrefLabel: "Financial Times", ID: mapper.IDURL(BrandFTUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Financial Times"},
		NarrowerConcepts: []Thing{
			{ID: mapper.IDURL(BrandLexUUID), APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), Types: typesUris, PrefLabel: "Lex", DirectType: typesUris[len(typesUris)-1], Predicate: skosNarrowerURI},
		},
	}

	expectedLex := Concept{APIURL: mapper.APIURL(BrandLexUUID, types, "Prod"), PrefLabel: "Lex", ID: mapper.IDURL(BrandLexUUID),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"LEX", "Lex"},
		NarrowerConcepts: []Thing{{ID: mapper.IDURL(BrandLexLiveUUID), APIURL: mapper.APIURL(BrandLexLiveUUID, types, "Prod"), Types: typesUris, PrefLabel: "Lex Live", DirectType: typesUris[len(typesUris)-1], Predicate: skosNarrowerURI}},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{narrower}
	actualFT, found, err := thingsDriver.read(BrandFTUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandFTUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandFTUUID)

	readAndCompare(t, expectedFT, actualFT, "Retrieve concepts via new concordance model")

	actualLex, found, err := thingsDriver.read(BrandLexUUID, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", BrandLexUUID)
	assert.True(t, found, "Found no Concept for concept as Concept %s", BrandLexUUID)

	readAndCompare(t, expectedLex, actualLex, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptWithRelated(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedOnyxPike := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		RelatedConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeRelated), APIURL: mapper.APIURL(TopicOnyxPikeRelated, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Related", DirectType: typesUris[len(typesUris)-1], Predicate: skosRelatedURI},
		},
	}

	expectedOnyxPikeBroader := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), PrefLabel: "Onyx Pike Broader", ID: mapper.IDURL(TopicOnyxPikeBroader),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Business & Economy", "Onyx Pike Broader"},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{related}
	actualOnyxPike, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedOnyxPike, actualOnyxPike, "Retrieve concepts via new concordance model")

	actualOnyxPikeBroader, found, err := thingsDriver.read(TopicOnyxPikeBroader, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPikeBroader)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPikeBroader)

	readAndCompare(t, expectedOnyxPikeBroader, actualOnyxPikeBroader, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptWithAllRelationships(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedOnyxPike := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		RelatedConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeRelated), APIURL: mapper.APIURL(TopicOnyxPikeRelated, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Related", DirectType: typesUris[len(typesUris)-1], Predicate: skosRelatedURI},
		},
		BroaderConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI},
			{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderTransitiveURI},
		},
	}

	expectedOnyxPikeBroader := Concept{APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), PrefLabel: "Onyx Pike Broader", ID: mapper.IDURL(TopicOnyxPikeBroader),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Onyx Pike Broader Business & Economy", "Onyx Pike Broader"},
		BroaderConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI},
		},
		NarrowerConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPike), APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike", DirectType: typesUris[len(typesUris)-1], Predicate: skosNarrowerURI},
		},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	relationships := []string{related, broader, broaderTransitive, narrower}
	actualOnyxPike, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedOnyxPike, actualOnyxPike, "Retrieve concepts via new concordance model")

	actualOnyxPikeBroader, found, err := thingsDriver.read(TopicOnyxPikeBroader, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPikeBroader)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPikeBroader)

	readAndCompare(t, expectedOnyxPikeBroader, actualOnyxPikeBroader, "Retrieve concepts via new concordance model")
}

func TestRetrieveConceptAsThingWithNotExistingRelationships(t *testing.T) {
	createOnyxPikeScenario(t)
	defer cleanUpOnyxPikeScenario(t)

	types := []string{"Thing", "Concept", "Topic"}
	typesUris := mapper.TypeURIs(types)

	expectedWithoutRelationships := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
	}

	expectedWithRelationships := Concept{APIURL: mapper.APIURL(TopicOnyxPike, types, "Prod"), PrefLabel: "Onyx Pike", ID: mapper.IDURL(TopicOnyxPike),
		Types: typesUris, DirectType: typesUris[len(typesUris)-1], Aliases: []string{"Bob", "BOB2"},
		DescriptionXML: "<p>Some stuff</p>", ImageURL: "http://media.ft.com/brand.png", EmailAddress: "email@email.com", ScopeNote: "bobs scopey notey", ShortLabel: "Short Label", TwitterHandle: "bob@twitter.com", FacebookPage: "bob@facebook.com",
		RelatedConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeRelated), APIURL: mapper.APIURL(TopicOnyxPikeRelated, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Related", DirectType: typesUris[len(typesUris)-1], Predicate: skosRelatedURI},
		},
		BroaderConcepts: []Thing{
			{ID: mapper.IDURL(TopicOnyxPikeBroader), APIURL: mapper.APIURL(TopicOnyxPikeBroader, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderURI},
			{ID: mapper.IDURL(TopicOnyxPikeBroaderTransitive), APIURL: mapper.APIURL(TopicOnyxPikeBroaderTransitive, types, "Prod"), Types: typesUris, PrefLabel: "Onyx Pike Broader Transitive", DirectType: typesUris[len(typesUris)-1], Predicate: skosBroaderTransitiveURI},
		},
	}

	thingsDriver := NewCypherDriver(db, "prod")

	actualWithoutRelationships, found, err := thingsDriver.read(TopicOnyxPike, []string{"something-that-do-not-exist"})

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedWithoutRelationships, actualWithoutRelationships, "Retrieve concepts via new concordance model")

	relationships := []string{"something-that-do-not-exist", broaderTransitive, related, "something-else-that-does-not-exist"}

	actualWithRelationships, found, err := thingsDriver.read(TopicOnyxPike, relationships)

	assert.NoError(t, err, "Unexpected error for concept as thing %s", TopicOnyxPike)
	assert.True(t, found, "Found no Concept for concept as Concept %s", TopicOnyxPike)

	readAndCompare(t, expectedWithRelationships, actualWithRelationships, "Retrieve concepts via new concordance model")
}

func createOnyxPikeScenario(t *testing.T) {
	conceptsDriver := concepts.NewConceptService(db)
	assert.NoError(t, conceptsDriver.Initialise())

	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPikeRelated-%s.json", TopicOnyxPikeRelated))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPikeBroaderTransitive-%s.json", TopicOnyxPikeBroaderTransitive))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPikeBroader-%s.json", TopicOnyxPikeBroader))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Topic-OnyxPike-%s.json", TopicOnyxPike))
}

func cleanUpOnyxPikeScenario(t *testing.T) {
	cleanDB(t, TopicOnyxPikeRelated, TopicOnyxPikeBroaderTransitive, TopicOnyxPikeBroader, TopicOnyxPike)
}

func createLexLiveScenario(t *testing.T) {
	conceptsDriver := concepts.NewConceptService(db)
	assert.NoError(t, conceptsDriver.Initialise())

	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Brand-FT-%s.json", BrandFTUUID))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Brand-Lex-%s.json", BrandLexUUID))
	writeJSONToConceptsService(t, conceptsDriver, fmt.Sprintf("./fixtures/Brand-LexLive-%s.json", BrandLexLiveUUID))
}

func cleanUpLexLiveScenario(t *testing.T) {
	cleanDB(t, BrandFTUUID, BrandLexUUID, BrandLexLiveUUID)
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
