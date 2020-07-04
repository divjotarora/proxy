package command

import (
	"errors"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func newCursorValueFixer(cf compositeFixer) valueFixer {
	fixers := compositeFixer{
		"ns": valueFixerFunc(fixCursorNSValue),
	}
	for k, v := range cf {
		fixers[k] = v
	}

	return newDocumentValueFixer("cursor", fixers)
}

func fixCursorNSValue(val bsoncore.Value, dst bsoncore.Document) (bsoncore.Document, error) {
	ns, ok := val.StringValueOK()
	if !ok {
		return nil, errors.New("FOOBAR2")
	}

	fixedNS := ns[5:] // remove "fixed" prefix
	dst = bsoncore.AppendStringElement(dst, "ns", fixedNS)
	return dst, nil
}
