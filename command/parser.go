package command

import (
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	dbKey = "$db"
)

var (
	// noopDatabaseNames contains names of databases that should be proxied without fixing.
	noopDatabaseNames = map[string]struct{}{
		"admin": {},
	}
)

// FixerSet represents two fixers associated with a command: one for the incoming request to the underlying server and
// one for the outgoing response back to the client.
type FixerSet struct {
	requestFixer  Fixer
	responseFixer Fixer
}

// FixRequest calls the registered Fixer for the incoming request to the underlying server.
func (f FixerSet) FixRequest(request bsoncore.Document) (bsoncore.Document, error) {
	return f.requestFixer.Fix(request)
}

// FixResponse calls the registered Fixer for the outgoing response back to the client.
func (f FixerSet) FixResponse(response bsoncore.Document) (bsoncore.Document, error) {
	return f.responseFixer.Fix(response)
}

// Parser parsers command names and maps them to Fixer implementations.
type Parser struct {
	fixers               map[string]FixerSet
	defaultFixerSet      FixerSet
	defaultRequestFixer  compositeFixer
	defaultResponseFixer Fixer
}

// NewParser initializes a new Parser instance.
func NewParser() *Parser {
	p := &Parser{
		fixers: make(map[string]FixerSet),
	}
	p.defaultRequestFixer = compositeFixer{
		dbKey: p.databaseNameValueFixer,
	}
	p.defaultResponseFixer = fixerFunc(noopFixer)
	p.defaultFixerSet = FixerSet{
		requestFixer:  p.defaultRequestFixer,
		responseFixer: p.defaultResponseFixer,
	}

	attachFixers(p)
	return p
}

// Parse returns the FixerSet for the given command.
func (p *Parser) Parse(cmdName string) FixerSet {
	if fixerSet, ok := p.fixers[cmdName]; ok {
		return fixerSet
	}
	return p.defaultFixerSet
}

func (p *Parser) register(cmdName string, requestFixer compositeFixer, responseFixer Fixer) {
	fullRequestFixer := compositeFixer{
		dbKey: p.databaseNameValueFixer,
	}
	for k, v := range requestFixer {
		fullRequestFixer[k] = v
	}

	if responseFixer == nil {
		responseFixer = p.defaultResponseFixer
	}
	set := FixerSet{
		requestFixer:  fullRequestFixer,
		responseFixer: responseFixer,
	}
	p.fixers[cmdName] = set
}

// valueFixer implementation for the $db value in a document.
func (p *Parser) databaseNameValueFixer(val bsoncore.Value, dst bsoncore.Document) (bsoncore.Document, error) {
	db, ok := val.StringValueOK()
	if !ok {
		return nil, fmt.Errorf("expected $db value to be string, got %s", val.Type)
	}

	fixedDB := db
	if _, ok := noopDatabaseNames[db]; !ok {
		fixedDB = fmt.Sprintf("fixed%s", db)
	}
	dst = bsoncore.AppendStringElement(dst, dbKey, fixedDB)
	return dst, nil
}

func cursorValueFixer(val bsoncore.Value, dst bsoncore.Document) (bsoncore.Document, error) {
	cursor, ok := val.DocumentOK()
	if !ok {
		return nil, fmt.Errorf("expected cursor value to be a document, got %s", val.Type)
	}

	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursor)
	return dst, nil
}
