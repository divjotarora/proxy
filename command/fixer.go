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

type valueFixer func(bsoncore.Value, bsoncore.Document) (bsoncore.Document, error)

type compositeFixer map[string]valueFixer

var _ Fixer = compositeFixer(nil)

func (cf compositeFixer) Fix(doc bsoncore.Document) (bsoncore.Document, error) {
	elems, err := doc.Elements()
	if err != nil {
		return nil, err
	}

	idx, fixed := bsoncore.AppendDocumentStart(nil)
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		vf, ok := cf[key]
		if !ok {
			fixed = bsoncore.AppendValueElement(fixed, key, elem.Value())
			continue
		}

		fixed, err = vf(val, fixed)
		if err != nil {
			return nil, err
		}
	}

	fixed, _ = bsoncore.AppendDocumentEnd(fixed, idx)
	return fixed, nil
}
