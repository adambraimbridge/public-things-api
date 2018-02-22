package things

import (
	"errors"
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	log "github.com/sirupsen/logrus"
)

const skosBroaderURI = "http://www.w3.org/2004/02/skos/core#broader"
const skosBroaderTransitiveURI = "http://www.w3.org/2004/02/skos/core#broaderTransitive"
const skosNarrowerURI = "http://www.w3.org/2004/02/skos/core#narrower"
const skosRelatedURI = "http://www.w3.org/2004/02/skos/core#related"

// Driver interface
type driver interface {
	read(id string, relationships []string) (thng Concept, found bool, err error)
	checkConnectivity() error
}

// CypherDriver struct
type cypherDriver struct {
	conn neoutils.NeoConnection
	env  string
}

func NewCypherDriver(conn neoutils.NeoConnection, env string) cypherDriver {
	return cypherDriver{conn, env}
}

func (cd cypherDriver) checkConnectivity() error { //TODO - use the neo4j connectivity check library
	return neoutils.Check(cd.conn)
}

type neoConcept struct {
	LeafUUID           string   `json:"leafUUID"`
	LeafPrefLabel      string   `json:"leafPrefLabel,omitempty"`
	LeafTypes          []string `json:"leafTypes"`
	LeafAliases        []string `json:"leafAliases,omitempty"`
	LeafDescriptionXML string   `json:"leafDescriptionXML,omitempty"`
	LeafImageURL       string   `json:"leafImageUrl,omitempty"`
	LeafEmailAddress   string   `json:"leafEmailAddress,omitempty"`
	LeafFacebookPage   string   `json:"leafFacebookPage,omitempty"`
	LeafTwitterHandle  string   `json:"leafTwitterHandle,omitempty"`
	LeafScopeNote      string   `json:"leafScopeNote,omitempty"`
	LeafShortLabel     string   `json:"leafShortLabel,omitempty"`

	CanonicalUUID           string   `json:"canonicalUUID"`
	CanonicalPrefLabel      string   `json:"canonicalPrefLabel,omitempty"`
	CanonicalTypes          []string `json:"canonicalTypes"`
	CanonicalAliases        []string `json:"canonicalAliases,omitempty"`
	CanonicalDescriptionXML string   `json:"canonicalDescriptionXML,omitempty"`
	CanonicalImageURL       string   `json:"canonicalImageUrl,omitempty"`
	CanonicalEmailAddress   string   `json:"canonicalEmailAddress,omitempty"`
	CanonicalFacebookPage   string   `json:"canonicalFacebookPage,omitempty"`
	CanonicalTwitterHandle  string   `json:"canonicalTwitterHandle,omitempty"`
	CanonicalScopeNote      string   `json:"canonicalScopeNote,omitempty"`
	CanonicalShortLabel     string   `json:"canonicalShortLabel,omitempty"`

	NarrowerConcepts          []neoThing `json:"narrowerConcepts,omitempty"`
	BroaderConcepts           []neoThing `json:"broaderConcepts,omitempty"`
	BroaderTransitiveConcepts []neoThing `json:"broaderTransitiveConcepts,omitempty"`
	RelatedConcepts           []neoThing `json:"relatedConcepts,omitempty"`
}

type neoThing struct {
	ID        string   `json:"id,omitempty"`
	PrefLabel string   `json:"prefLabel,omitempty"`
	Types     []string `json:"types,omitempty"`
}

func (cd cypherDriver) read(thingUUID string, relationships []string) (Concept, bool, error) {
	var results []neoConcept

	cypherStmt := newCypherStmtBuilder().withRelationships(relationships).build()

	query := &neoism.CypherQuery{
		Statement:  cypherStmt,
		Parameters: neoism.Props{"thingUUID": thingUUID},
		Result:     &results,
	}
	log.Debugf("Query: %v", query)
	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil || len(results) == 0 || len(results[0].LeafUUID) == 0 {
		return Concept{}, false, err
	} else if len(results) != 1 && len(results[0].LeafUUID) != 1 {
		errMsg := fmt.Sprintf("Multiple Things found with the same UUID:%s !", thingUUID)
		log.WithFields(log.Fields{"UUID": thingUUID}).Error("Multiple Things found with the same UUID")
		return Concept{}, true, errors.New(errMsg)
	} else if isContent(results[0]) {
		return Concept{}, false, nil
	} else {
		thing, err := mapToResponseFormat(results[0], cd.env)
		return thing, true, err
	}

}

func isContent(thng neoConcept) bool {
	for _, label := range thng.LeafTypes {
		if label == "Content" {
			return true
		}
	}
	return false
}

func mapToResponseFormat(thng neoConcept, env string) (Concept, error) {
	log.Debugf("NeoConcept: %v", thng)
	thing := Concept{}

	// New Concordance Model
	if thng.CanonicalPrefLabel != "" {
		thing.PrefLabel = thng.CanonicalPrefLabel
		thing.APIURL = mapper.APIURL(thng.CanonicalUUID, thng.CanonicalTypes, env)
		thing.ID = mapper.IDURL(thng.CanonicalUUID)
		types := mapper.TypeURIs(thng.CanonicalTypes)
		if types == nil {
			log.WithFields(log.Fields{"UUID": thng.CanonicalUUID}).Errorf("Could not map type URIs for ID %s with types %s", thng.CanonicalUUID, thng.CanonicalTypes)
			return thing, errors.New("concept not found")
		}
		thing.Types = types
		thing.DirectType = types[len(types)-1]
		thing.DescriptionXML = thng.CanonicalDescriptionXML
		thing.Aliases = thng.CanonicalAliases
		thing.ImageURL = thng.CanonicalImageURL
		thing.EmailAddress = thng.CanonicalEmailAddress
		thing.TwitterHandle = thng.CanonicalTwitterHandle
		thing.FacebookPage = thng.CanonicalFacebookPage
		thing.ScopeNote = thng.CanonicalScopeNote
		thing.ShortLabel = thng.CanonicalShortLabel

	} else {
		thing.PrefLabel = thng.LeafPrefLabel
		thing.APIURL = mapper.APIURL(thng.LeafUUID, thng.LeafTypes, env)
		thing.ID = mapper.IDURL(thng.LeafUUID)
		types := mapper.TypeURIs(thng.LeafTypes)
		if types == nil {
			log.WithFields(log.Fields{"UUID": thng.LeafUUID}).Errorf("Could not map type URIs for ID %s with types %s", thng.LeafUUID, thng.LeafTypes)
			return thing, errors.New("concept not found")
		}
		thing.Types = types
		thing.DirectType = types[len(types)-1]
		thing.DescriptionXML = thng.LeafDescriptionXML
		thing.Aliases = thng.LeafAliases
		thing.ImageURL = thng.LeafImageURL
		thing.EmailAddress = thng.LeafEmailAddress
		thing.TwitterHandle = thng.LeafTwitterHandle
		thing.FacebookPage = thng.LeafFacebookPage
		thing.ScopeNote = thng.LeafScopeNote
		thing.ShortLabel = thng.LeafShortLabel
	}

	thing.BroaderConcepts = populateRelationships(thng.BroaderConcepts, skosBroaderURI, thng.BroaderTransitiveConcepts, skosBroaderTransitiveURI, env)
	thing.NarrowerConcepts = populateRelationships(thng.NarrowerConcepts, skosNarrowerURI, nil, "", env)
	thing.RelatedConcepts = populateRelationships(thng.RelatedConcepts, skosRelatedURI, nil, "", env)

	log.Debugf("Mapped Concept: %v", thing)
	return thing, nil
}

func populateRelationships(concepts []neoThing, predicate string, transitiveConcepts []neoThing, transitivePredicate string, env string) []Thing {
	if len(concepts) > 0 && concepts[0].ID != "" {
		var things []Thing
		directConceptCache := make(map[string]struct{})
		for _, c := range concepts {
			directConceptCache[c.ID] = struct{}{}
			t := mapToThingInRelationship(c, env, predicate)
			things = append(things, t)
		}
		if len(transitiveConcepts) > 0 && transitiveConcepts[0].ID != "" {
			for _, tc := range transitiveConcepts {
				if _, found := directConceptCache[tc.ID]; !found {
					t := mapToThingInRelationship(tc, env, transitivePredicate)
					things = append(things, t)
				}
			}
		}
		return things
	}
	return nil
}

func mapToThingInRelationship(c neoThing, env, predicate string) Thing {
	var t Thing
	brTypes := mapper.TypeURIs(c.Types)
	if brTypes != nil {
		t.PrefLabel = c.PrefLabel
		t.APIURL = mapper.APIURL(c.ID, c.Types, env)
		t.ID = mapper.IDURL(c.ID)
		t.Types = brTypes
		t.DirectType = brTypes[len(brTypes)-1]
		t.Predicate = predicate
	} else {
		log.WithFields(log.Fields{"UUID": c.ID}).Errorf("Could not map type URIs for ID %s with types %s", c.ID, c.Types)
	}
	return t
}
