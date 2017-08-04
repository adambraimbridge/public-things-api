package things

import (
	"errors"
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type driver interface {
	read(id string) (thng Concept, found bool, err error)
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

type neoThing struct {
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
}

func (cd cypherDriver) read(thingUUID string) (Concept, bool, error) {
	results := []neoThing{}

	// This is just getting the broader than at the moment

	//MATCH (identifier:UPPIdentifier{value:"2faae810-517d-4b6d-bb7b-1df10dcbe243"})
	//MATCH (identifier)-[:IDENTIFIES]->(leaf:Concept)
	//OPTIONAL MATCH (leaf)-[:EQUIVALENT_TO]->(canonical:Concept)
	//OPTIONAL MATCH (leaf)-[:HAS_BROADER]->(broader:Concept)
	//WITH leaf, canonical, {uuid: broader.uuid, prefLabel: broader.prefLabel, types: labels(broader)} as b
	//OPTIONAL MATCH (leaf)<-[:HAS_BROADER]-(narrower:Concept)
	//WITH leaf, canonical, collect(b) as broader, {uuid: narrower.uuid, prefLabel: narrower.prefLabel, types: labels(narrower)} as n
	//RETURN leaf.uuid as leafUUID, broader, collect(n) as narrower

	//MATCH (identifier:UPPIdentifier{value:"2faae810-517d-4b6d-bb7b-1df10dcbe243"})
	//MATCH (identifier)-[:IDENTIFIES]->(leaf:Concept)
	//OPTIONAL MATCH (leaf)-[:EQUIVALENT_TO]->(canonical:Concept)
	//OPTIONAL MATCH (leaf)-[:HAS_BROADER]->(broader:Concept)
	//WITH leaf, canonical, {uuid: broader.uuid, prefLabel: broader.prefLabel, types: labels(broader)} as broader
	//RETURN leaf.uuid as leafUUID, labels(leaf) as leafTypes, leaf.prefLabel as leafPrefLabel,
	//	leaf.descriptionXML as leafDescriptionXML, leaf.imageUrl as leafImageUrl, leaf.aliases as leafAliases, leaf.emailAddress as leafEmailAddress,
	//	leaf.facebookPage as leafFacebookPage, leaf.twitterHandle as leafTwitterHandle, leaf.scopeNote as leafScopeNote, leaf.shortLabel as leafShortLabel,
	//	canonical.prefUUID as canonicalUUID, canonical.prefLabel as canonicalPrefLabel, labels(canonical) as canonicalTypes,
	//	canonical.descriptionXML as canonicalDescriptionXML, canonical.imageUrl as canonicalImageUrl, canonical.aliases as canonicalAliases, canonical.emailAddress as canonicalEmailAddress,
	//	canonical.facebookPage as canonicalFacebookPage, canonical.twitterHandle as canonicalTwitterHandle, canonical.scopeNote as canonicalScopeNote, canonical.shortLabel as canonicalShortLabel, collect(broader) as bboo

	query := &neoism.CypherQuery{
		Statement: `MATCH (identifier:UPPIdentifier{value:{thingUUID}})
 			MATCH (identifier)-[:IDENTIFIES]->(leaf:Concept)
 			OPTIONAL MATCH (leaf)-[:EQUIVALENT_TO]->(canonical:Concept)
 			OPTIONAL MATCH (leaf)-[:HAS_BROADER]->(broader:Concept)
 			WITH leaf, canonical, {uuid: broader.uuid, prefLabel: broader.prefLabel} as broader
 			OPTIONAL MATCH (leaf)<-[:HAS_BROADER]-(narrower:Concept)
 			OPTIONAL MATCH (leaf)-[:RELATED_TO]-(relatedto:Concept)
			RETURN leaf.uuid as leafUUID, labels(leaf) as leafTypes, leaf.prefLabel as leafPrefLabel,
			leaf.descriptionXML as leafDescriptionXML, leaf.imageUrl as leafImageUrl, leaf.aliases as leafAliases, leaf.emailAddress as leafEmailAddress,
			leaf.facebookPage as leafFacebookPage, leaf.twitterHandle as leafTwitterHandle, leaf.scopeNote as leafScopeNote, leaf.shortLabel as leafShortLabel,
			canonical.prefUUID as canonicalUUID, canonical.prefLabel as canonicalPrefLabel, labels(canonical) as canonicalTypes,
			canonical.descriptionXML as canonicalDescriptionXML, canonical.imageUrl as canonicalImageUrl, canonical.aliases as canonicalAliases, canonical.emailAddress as canonicalEmailAddress,
			canonical.facebookPage as canonicalFacebookPage, canonical.twitterHandle as canonicalTwitterHandle, canonical.scopeNote as canonicalScopeNote, canonical.shortLabel as canonicalShortLabel
			`,

		Parameters: neoism.Props{"thingUUID": thingUUID},
		Result:     &results,
	}

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

func isContent(thng neoThing) bool {
	for _, label := range thng.LeafTypes {
		if label == "Content" {
			return true
		}
	}
	return false
}

func mapToResponseFormat(thng neoThing, env string) (Concept, error) {
	thing := Concept{}
	// New Concordance Model
	if thng.CanonicalPrefLabel != "" {
		thing.PrefLabel = thng.CanonicalPrefLabel
		thing.APIURL = mapper.APIURL(thng.CanonicalUUID, thng.CanonicalTypes, env)
		thing.ID = mapper.IDURL(thng.CanonicalUUID)
		types := mapper.TypeURIs(thng.CanonicalTypes)
		if types == nil {
			log.WithFields(log.Fields{"UUID": thng.CanonicalUUID}).Errorf("Could not map type URIs for ID %s with types %s", thng.CanonicalUUID, thng.CanonicalTypes)
			return thing, errors.New("Concept not found")
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
			return thing, errors.New("Concept not found")
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
	return thing, nil
}
