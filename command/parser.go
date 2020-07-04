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
	fixers map[string]Fixer
}

// NewParser initializes a new Parser instance.
func NewParser() *Parser {
	p := &Parser{
		fixers: make(map[string]Fixer),
	}

	return p
}

// Parse returns the Fixer for the given command.
func (p *Parser) Parse(cmdName string) Fixer {
	_, ok := p.fixers[cmdName]
	if ok {
		panic("not implemented")
	}

	return fixerFunc(p.databaseNameFixer)
}

// POC for a Fixer implementation that will replace the $db value in the command document with "fixed<dbValue>"
func (p *Parser) databaseNameFixer(cmd bsoncore.Document) (bsoncore.Document, error) {
	elems, err := cmd.Elements()
	if err != nil {
		return nil, err
	}

	idx, fixed := bsoncore.AppendDocumentStart(nil)
	for _, elem := range elems {
		if key := elem.Key(); key != dbKey {
			fixed = bsoncore.AppendValueElement(fixed, key, elem.Value())
			continue
		}

		val := elem.Value()
		db, ok := val.StringValueOK()
		if !ok {
			return nil, fmt.Errorf("expected $db value to string, got %s", val.Type)
		}

		fixedDB := db
		if _, ok := noopDatabaseNames[db]; !ok {
			fixedDB = fmt.Sprintf("fixed%s", db)
		}
		fixed = bsoncore.AppendStringElement(fixed, dbKey, fixedDB)
	}

	fixed, err = bsoncore.AppendDocumentEnd(fixed, idx)
	if err != nil {
		return nil, err
	}
	return fixed, nil
}
