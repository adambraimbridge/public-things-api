package things

import (
	"fmt"
	"regexp"
)

const thingMatchStatements = `MATCH (identifier:UPPIdentifier{value:{thingUUID}})
 							  MATCH (identifier)-[:IDENTIFIES]->(leaf:Concept)
                              OPTIONAL MATCH (leaf)-[:EQUIVALENT_TO]->(canonical:Concept) `
const relationshipsMatchStatementsTemplate = `OPTIONAL MATCH (leaf)%s(c%v:Concept)
                                              OPTIONAL MATCH (c%v)-[:EQUIVALENT_TO]->(%sCanonical:Concept) `
const conceptMapTemplate = ", {id: %sCanonical.prefUUID, prefLabel: %sCanonical.prefLabel, types: labels(%sCanonical)} as %sMap "
const conceptCollectionTemplate = ", collect(DISTINCT %sMap) as %sConcepts"
const thingReturnStatement = `RETURN 
                              leaf.uuid as leafUUID, labels(leaf) as leafTypes, leaf.prefLabel as leafPrefLabel,
                              leaf.descriptionXML as leafDescriptionXML, leaf.imageUrl as leafImageUrl, leaf.aliases as leafAliases, leaf.emailAddress as leafEmailAddress,
                              leaf.facebookPage as leafFacebookPage, leaf.twitterHandle as leafTwitterHandle, leaf.scopeNote as leafScopeNote, leaf.shortLabel as leafShortLabel,
                              canonical.prefUUID as canonicalUUID, canonical.prefLabel as canonicalPrefLabel, labels(canonical) as canonicalTypes,
                              canonical.descriptionXML as canonicalDescriptionXML, canonical.imageUrl as canonicalImageUrl, canonical.aliases as canonicalAliases, canonical.emailAddress as canonicalEmailAddress,
                              canonical.facebookPage as canonicalFacebookPage, canonical.twitterHandle as canonicalTwitterHandle, canonical.scopeNote as canonicalScopeNote, canonical.shortLabel as canonicalShortLabel`

var skosNeo4JRelationshipMap = map[string]string{
	"broader":           "-[:HAS_BROADER]->",
	"broaderTransitive": "-[:HAS_BROADER*2..]->",
	"narrower":          "<-[:HAS_BROADER]-",
	"related":           "-[:IS_RELATED_TO]->",
}

var collectStmtRegExp = regexp.MustCompile("collect\\(DISTINCT \\w+Map\\) as ")

type cypherStmtBuilder struct {
	thingUUID     string
	relationships []string
}

func newCypherStmtBuilder() *cypherStmtBuilder {
	return &cypherStmtBuilder{}
}

func (b *cypherStmtBuilder) withRelationships(relationships []string) *cypherStmtBuilder {
	b.relationships = relationships
	return b
}

func (b *cypherStmtBuilder) build() string {
	stmt := thingMatchStatements
	stmt += buildRelationshipsMatchStatements(b.relationships)
	stmt += buildReturnStatement(b.relationships)
	return stmt
}

func buildRelationshipsMatchStatements(relationships []string) string {
	stmt := ""
	previousRelationship := ""
	withStmt := "WITH leaf, canonical"
	for i, r := range relationships {
		stmt += fmt.Sprintf(relationshipsMatchStatementsTemplate, skosNeo4JRelationshipMap[r], i, i, r)

		withStmt = updatedWithStmt(withStmt, previousRelationship)

		stmt += withStmt + fmt.Sprintf(conceptMapTemplate, r, r, r, r)

		previousRelationship = r
	}

	if len(relationships) > 0 {
		withStmt = updatedWithStmt(withStmt, previousRelationship)
		stmt += withStmt + " "
	}

	return stmt
}

func buildReturnStatement(relationships []string) string {
	stmt := thingReturnStatement
	for _, r := range relationships {
		stmt += ", " + r + "Concepts"
	}
	return stmt
}

func updatedWithStmt(withStmt, relationship string) string {
	if relationship != "" {
		withStmt = collectStmtRegExp.ReplaceAllString(withStmt, "")
		withStmt += fmt.Sprintf(conceptCollectionTemplate, relationship, relationship)
	}
	return withStmt
}
