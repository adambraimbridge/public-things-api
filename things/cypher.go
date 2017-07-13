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
	read(id string) (thng thing, found bool, err error)
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
	CanonicalUUID      string   `json:"canonicalUUID"`
	CanonicalPrefLabel string   `json:"canonicalPrefLabel,omitempty"`
	CanonicalTypes     []string `json:"canonicalTypes"`
}

func (cd cypherDriver) read(thingUUID string) (thing, bool, error) {
	results := []neoThing{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (identifier:UPPIdentifier{value:{thingUUID}})
 			MATCH (identifier)-[:IDENTIFIES]->(leaf:Thing)
 			OPTIONAL MATCH (leaf)-[:EQUIVALENT_TO]->(canonical:Thing)
			RETURN leaf.uuid as leafUUID, labels(leaf) as leafTypes, leaf.prefLabel as leafPrefLabel,
			canonical.prefUUID as canonicalUUID, canonical.prefLabel as canonicalPrefLabel, labels(canonical) as canonicalTypes `,
		Parameters: neoism.Props{"thingUUID": thingUUID},
		Result:     &results,
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil || len(results) == 0 || len(results[0].LeafUUID) == 0 {
		return thing{}, false, err
	} else if len(results) != 1 && len(results[0].LeafUUID) != 1 {
		errMsg := fmt.Sprintf("Multiple things found with the same UUID:%s !", thingUUID)
		log.WithFields(log.Fields{"UUID": thingUUID}).Error("Multiple things found with the same UUID")
		return thing{}, true, errors.New(errMsg)
	} else if isContent(results[0]) {
		return thing{}, false, nil
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

func mapToResponseFormat(thng neoThing, env string) (thing, error) {
	thing := thing{}
	// New Concordance Model
	if thng.CanonicalPrefLabel != "" {
		thing.PrefLabel = thng.CanonicalPrefLabel
		thing.APIURL = mapper.APIURL(thng.CanonicalUUID, thng.CanonicalTypes, env)
		thing.ID = mapper.IDURL(thng.CanonicalUUID)
		types := mapper.TypeURIs(thng.CanonicalTypes)
		if types == nil {
			log.WithFields(log.Fields{"UUID": thng.CanonicalUUID}).Errorf("Could not map type URIs for ID %s with types %s", thng.CanonicalUUID, thng.CanonicalTypes)
			return thing, errors.New("Thing not found")
		}
		thing.Types = types
		thing.DirectType = types[len(types)-1]
	} else {
		thing.PrefLabel = thng.LeafPrefLabel
		thing.APIURL = mapper.APIURL(thng.LeafUUID, thng.LeafTypes, env)
		thing.ID = mapper.IDURL(thng.LeafUUID)
		types := mapper.TypeURIs(thng.LeafTypes)
		if types == nil {
			log.WithFields(log.Fields{"UUID": thng.LeafUUID}).Errorf("Could not map type URIs for ID %s with types %s", thng.LeafUUID, thng.LeafTypes)
			return thing, errors.New("Thing not found")
		}
		thing.Types = types
		thing.DirectType = types[len(types)-1]
	}
	return thing, nil
}
