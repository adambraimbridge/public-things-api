package things

import (
	"fmt"

	"errors"

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

type neoReadStruct struct {
}

func (cd cypherDriver) read(thingUUID string) (thing, bool, error) {
	results := []thing{}

	query := &neoism.CypherQuery{
		Statement: `
					MATCH (identifier:UPPIdentifier{value:{thingUUID}})
 					MATCH (identifier)-[:IDENTIFIES]->(c:Thing)
					RETURN c.uuid as id,
					labels(c) as types,
					c.prefLabel as prefLabel
					`,
		Parameters: neoism.Props{"thingUUID": thingUUID},
		Result:     &results,
	}

	if err := cd.conn.CypherBatch([]*neoism.CypherQuery{query}); err != nil || len(results) == 0 || len(results[0].ID) == 0 {
		return thing{}, false, err
	} else if len(results) != 1 && len(results[0].ID) != 1 {
		errMsg := fmt.Sprintf("Multiple things found with the same uuid:%s !", thingUUID)
		log.Error(errMsg)
		return thing{}, true, errors.New(errMsg)
	} else if isContent(results[0]) {
		return thing{}, false, nil
	} else {
		thing, err := mapToResponseFormat(&results[0], cd.env)
		return *thing, true, err
	}
}

func isContent(thng thing) bool {
	for _, label := range thng.Types {
		if label == "Content" {
			return true
		}
	}
	return false
}

func mapToResponseFormat(thng *thing, env string) (*thing, error) {
	thng.APIURL = mapper.APIURL(thng.ID, thng.Types, env)
	thng.ID = mapper.IDURL(thng.ID)
	types := mapper.TypeURIs(thng.Types)
	if types == nil {
		log.Errorf("Could not map type URIs for ID %s with types %s", thng.ID, thng.Types)
		return thng, errors.New("Thing not found")
	}
	thng.Types = types
	thng.DirectType = types[len(types)-1]
	return thng, nil
}
