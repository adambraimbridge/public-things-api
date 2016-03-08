package main

import (
	"fmt"

	"errors"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
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
	db  *neoism.Database
	env string
}

func newCypherDriver(db *neoism.Database, env string) cypherDriver {
	return cypherDriver{db, env}
}

func (cd cypherDriver) checkConnectivity() error { //TODO - use the neo4j connectivity check library
	results := []struct {
		ID int
	}{}
	query := &neoism.CypherQuery{
		Statement: "MATCH (x) RETURN ID(x) LIMIT 1",
		Result:    &results,
	}
	err := cd.db.Cypher(query)
	log.Debugf("CheckConnectivity results:%+v  err: %+v", results, err)
	return err
}

type neoReadStruct struct {
}

func (cd cypherDriver) read(thingUUID string) (thing, bool, error) {
	results := []thing{}

	query := &neoism.CypherQuery{
		Statement: `
					MATCH (c:Thing{uuid:{thingUUID}})
					RETURN c.uuid as id,
					labels(c) as types,
					c.prefLabel as prefLabel
					`,
		Parameters: neoism.Props{"thingUUID": thingUUID},
		Result:     &results,
	}
	err := cd.db.Cypher(query)
	if err != nil {
		log.Errorf("Error looking up uuid %s with query %s from neoism: %+v", thingUUID, query.Statement, err)
		return thing{}, false, fmt.Errorf("Error accessing Things datastore for uuid: %s", thingUUID)
	}
	log.Debugf("Found %d Things for uuid: %s", len(results), thingUUID)
	if (len(results)) == 0 {
		return thing{}, false, nil
	}

	if (len(results)) > 1 {
		return thing{}, true, errors.New(fmt.Sprintf("Multiple things for %v", thingUUID))
	}

	thng, err := mapToResponseFormat(&results[0], cd.env)
	return *thng, true, err
}

func mapToResponseFormat(thng *thing, env string) (*thing, error) {
	thng.APIURL = mapper.APIURL(thng.ID, thng.Types, env)
	thng.ID = mapper.IDURL(thng.ID)
	types := mapper.TypeURIs(thng.Types) //TODO - we should decide which types should be exposed
	if types == nil {
		log.Warnf("Could not map type URIs for ID %s with types %s", thng.ID, thng.Types)
		return thng, errors.New("Concept not found")
	}
	thng.Types = types
	thng.DirectType = thng.Types[0] //TODO - use mapper when the call is complete
	return thng, nil
}

