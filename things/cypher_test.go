package things

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/memberships-rw-neo4j/memberships"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/roles-rw-neo4j/roles"
	"github.com/Financial-Times/subjects-rw-neo4j/subjects"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	//Generate uuids so there's no clash with real data
	FakebookConceptUUID        = "eac853f5-3859-4c08-8540-55e043719400" //organization
	MetalMickeyConceptUUID     = "0483bef8-5797-40b8-9b25-b12e492f63c6" //subject
	ContentUUID                = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	RoleUUID                   = "4f01dce1-142d-4ebf-b73b-587086cce0f9"
	BoardRoleUUID              = "2f91f554-0eb0-4ee6-9856-7561bf925d74"
	MembershipUUID             = "c8e19a44-a323-4ce0-b76b-6b23f6c7e2a5"
	MembershipRoleUUID         = "3d7e102d-14b9-42d5-b20e-7b9fd497f405"
	MembershipPersonUUID       = "d00dc7f6-6f40-4350-bf72-37c4253f3d7c"
	MembershipOrganisationUUID = "778a9149-2097-4a69-9a28-e0a782bdc1a4"
	NonExistingThingUUID       = "b2860919-4b78-44c6-a665-af9221bdefb5"
)

func TestRetrieveOrganizationAsThing(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnection(t, assert)

	organisationRW := writeOrganisation(assert, db)

	defer deleteOrganisation(organisationRW)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(FakebookConceptUUID)
	assert.NoError(err, "Unexpected error for organisation as thing %s", FakebookConceptUUID)
	assert.True(found, "Found no thing for organisation as thing %s", FakebookConceptUUID)
	validateThing(assert, "Fakebook, Inc.", FakebookConceptUUID, "http://www.ft.com/ontology/company/PublicCompany",
		[]string{
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/organisation/Organisation",
			"http://www.ft.com/ontology/company/Company",
			"http://www.ft.com/ontology/company/PublicCompany",
		}, thng)
}

func TestRetrieveSubjectAsThing(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)
	subjectRW := writeSubject(assert, db)

	defer deleteSubject(subjectRW)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(MetalMickeyConceptUUID)
	assert.NoError(err, "Unexpected error for content %s", MetalMickeyConceptUUID)
	assert.True(found, "Found no thing for content %s", MetalMickeyConceptUUID)
	validateThing(assert, "Metal Mickey", MetalMickeyConceptUUID, "http://www.ft.com/ontology/Subject",
		[]string{
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/classification/Classification",
			"http://www.ft.com/ontology/Subject",
		}, thng)
}

func TestRetrieveMembershipAsThing(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)
	membershipRW := writeMembership(assert, db)

	defer deleteMembership(membershipRW)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(MembershipUUID)
	assert.NoError(err, "Unexpected error for membership %s", MembershipUUID)
	assert.True(found, "Found no thing for membership %s", MembershipUUID)
	validateThing(assert, "Market Strategist", MembershipUUID, "http://www.ft.com/ontology/organisation/Membership", []string{
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/concept/Concept",
		"http://www.ft.com/ontology/organisation/Membership",
	}, thng)
}

func TestRetrieveRolesAsThing(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)
	rolesRW := writeRoles(assert, db)

	defer deleteRoles(rolesRW)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(RoleUUID)
	assert.NoError(err, "Unexpected error for role %s", RoleUUID)
	assert.True(found, "Found no thing for role %s", RoleUUID)
	validateThing(assert, "Market Strategist", RoleUUID, "http://www.ft.com/ontology/organisation/Role", []string{
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/organisation/Role",
	}, thng)

	thng, found, err = thingsDriver.read(BoardRoleUUID)
	assert.NoError(err, "Unexpected error for content %s", BoardRoleUUID)
	validateThing(assert, "Chairman of the Board", BoardRoleUUID, "http://www.ft.com/ontology/organisation/BoardRole", []string{
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/organisation/Role",
		"http://www.ft.com/ontology/organisation/BoardRole",
	}, thng)
}

//TODO - this is temporary, we WILL want to retrieve Content once we have more info about it available
func TestCannotRetrieveContentAsThing(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)
	contentRW := writeContent(assert, db)

	defer deleteContent(contentRW)
	defer cleanUpBrandsUppIdentifier(db, t, assert)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(thing{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func TestRetrieveNoThingsWhenThereAreNonePresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	thingsDriver := NewCypherDriver(db, "prod")
	thng, found, err := thingsDriver.read(NonExistingThingUUID)
	assert.NoError(err, "Unexpected error for thing %s", NonExistingThingUUID)
	assert.False(found, "Found thing %s", NonExistingThingUUID)
	assert.EqualValues(thing{}, thng, "Found non-existing thing %s", NonExistingThingUUID)
}

func writeOrganisation(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())
	writeJSONToService(organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json", assert)
	return organisationRW
}

func deleteOrganisation(organisationRW baseftrwapp.Service) {
	organisationRW.Delete(FakebookConceptUUID)
}

func writeSubject(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	subjectsRW := subjects.NewCypherSubjectsService(db)
	assert.NoError(subjectsRW.Initialise())
	writeJSONToService(subjectsRW, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json", assert)
	return subjectsRW
}

func deleteSubject(subjectsRW baseftrwapp.Service) {
	subjectsRW.Delete(MetalMickeyConceptUUID)
}

func writeContent(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	contentRW := content.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-Bitcoin-3fc9fe3e-af8c-4f7f-961a-e5065392bb31.json", assert)
	return contentRW
}

func cleanUpBrandsUppIdentifier(db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			//deletes parent 'org' which only has type Thing
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) DETACH DELETE a", "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"),
		},
		{
			//deletes upp identifier for the above parent 'org'
			Statement: fmt.Sprintf("MATCH (b:Identifier {value: '%v'}) DETACH DELETE b", "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func deleteContent(contentRW baseftrwapp.Service) {
	contentRW.Delete(ContentUUID)
}

func writeMembership(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	membershipsRW := memberships.NewCypherMembershipService(db)
	assert.NoError(membershipsRW.Initialise())
	writeJSONToService(membershipsRW, "./fixtures/Membership-c8e19a44-a323-4ce0-b76b-6b23f6c7e2a5.json", assert)
	return membershipsRW
}

func deleteMembership(membershipsRW baseftrwapp.Service) {
	membershipsRW.Delete(MembershipUUID)
	membershipsRW.Delete(MembershipRoleUUID)
	membershipsRW.Delete(MembershipPersonUUID)
	membershipsRW.Delete(MembershipOrganisationUUID)
}

func writeRoles(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	rolesRW := roles.NewCypherDriver(db)
	assert.NoError(rolesRW.Initialise())
	writeJSONToService(rolesRW, "./fixtures/Role-MarketStrategist-4f01dce1-142d-4ebf-b73b-587086cce0f9.json", assert)
	writeJSONToService(rolesRW, "./fixtures/BoardRole-Chairman-2f91f554-0eb0-4ee6-9856-7561bf925d74.json", assert)
	return rolesRW
}

func deleteRoles(rolesRW baseftrwapp.Service) {
	rolesRW.Delete(RoleUUID)
	rolesRW.Delete(BoardRoleUUID)
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

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func validateThing(assert *assert.Assertions, prefLabel string, UUID string, directType string, types []string, thng thing) {
	assert.EqualValues(prefLabel, thng.PrefLabel, "PrefLabel incorrect")
	assert.EqualValues("http://api.ft.com/things/"+UUID, thng.ID, "ID incorrect")
	assert.EqualValues(directType, thng.DirectType, "DirectType incorrect")
	assert.EqualValues(types, thng.Types, "Types incorrect")
}
