package command

import (
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// FixerSet represents two fixers associated with a command: one for the incoming request to the underlying server and
// one for the outgoing response back to the client.
type FixerSet struct {
	requestFixer  DocumentFixer
	responseFixer DocumentFixer
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
	fixers          map[string]FixerSet
	defaultFixerSet FixerSet
}

// NewParser initializes a new Parser instance.
func NewParser() *Parser {
	p := &Parser{
		fixers: make(map[string]FixerSet),
	}
	p.defaultFixerSet = FixerSet{
		requestFixer:  p.createDefaultRequestFixer(),
		responseFixer: p.createDefaultResponseFixer(),
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

func (p *Parser) createDefaultRequestFixer() DocumentFixer {
	// By default, only the $db value is fixed in requests to prepend a prefix to the database name.
	return DocumentFixer{
		"$db": addDBPrefixValueFixer,
	}
}

func (p *Parser) createDefaultResponseFixer() DocumentFixer {
	return DocumentFixer{
		"writeErrors": writeErrorsValueFixer,
	}
}

func (p *Parser) register(cmdName string, requestFixer DocumentFixer, responseFixer DocumentFixer) {
	fullRequestFixer := p.createDefaultRequestFixer()
	for k, v := range requestFixer {
		fullRequestFixer[k] = v
	}

	fullResponseFixer := p.createDefaultResponseFixer()
	for k, v := range responseFixer {
		fullResponseFixer[k] = v
	}

	p.fixers[cmdName] = FixerSet{
		requestFixer:  fullRequestFixer,
		responseFixer: fullResponseFixer,
	}
}
