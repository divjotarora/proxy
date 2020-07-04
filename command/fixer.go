package command

import "go.mongodb.org/mongo-driver/x/bsonx/bsoncore"

// Fixer is responsible for fixing command and response documents to add or remove information when proxying messages.
type Fixer interface {
	Fix(bsoncore.Document) (bsoncore.Document, error)
}

// fixerFunc is an implementation of Fixer as a standalone function.
type fixerFunc func(bsoncore.Document) (bsoncore.Document, error)

var _ Fixer = fixerFunc(nil)

func (f fixerFunc) Fix(cmd bsoncore.Document) (bsoncore.Document, error) {
	return f(cmd)
}
