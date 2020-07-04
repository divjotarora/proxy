package command

import (
	"errors"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func newDefaultCursorValueFixer(batchDocsFixer valueFixer) valueFixer {
	fixers := compositeFixer{
		"ns": valueFixerFunc(fixCursorNSValue),
	}
	if batchDocsFixer != nil {
		avf := newArrayValueFixer(batchDocsFixer)
		fixers["firstBatch"] = avf
		fixers["nextBatch"] = avf
	}

	return newDocumentValueFixer(fixers)
}

func fixCursorNSValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	ns, ok := val.StringValueOK()
	if !ok {
		return nil, errors.New("FOOBAR2")
	}

	fixedNS := ns[5:] // remove "fixed" prefix
	dst = bsoncore.AppendStringElement(dst, key, fixedNS)
	return dst, nil
}
