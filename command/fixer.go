package command

import "go.mongodb.org/mongo-driver/x/bsonx/bsoncore"

type Fixer interface {
	Fix(bsoncore.Document) (bsoncore.Document, error)
}

type fixerFunc func(bsoncore.Document) (bsoncore.Document, error)

var _ Fixer = fixerFunc(nil)

func (f fixerFunc) Fix(cmd bsoncore.Document) (bsoncore.Document, error) {
	return f(cmd)
}
