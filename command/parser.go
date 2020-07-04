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

// Parser parsers command names and maps them to Fixer implementations.
type Parser struct {
	fixers       map[string]Fixer
	defaultFixer Fixer
}

// NewParser initializes a new Parser instance.
func NewParser() *Parser {
	p := &Parser{
		fixers: make(map[string]Fixer),
	}
	p.defaultFixer = compositeFixer{
		dbKey: p.databaseNameValueFixer,
	}

	return p
}

// Parse returns the Fixer for the given command.
func (p *Parser) Parse(cmdName string) Fixer {
	_, ok := p.fixers[cmdName]
	if ok {
		panic("not implemented")
	}

	return p.defaultFixer
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
