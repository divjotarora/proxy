package command

import (
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// type valueFixer func(bsoncore.Value, bsoncore.Document) (bsoncore.Document, error)
type valueFixer interface {
	fixValue(bsoncore.Value, bsoncore.Document) (bsoncore.Document, error)
}

type valueFixerFunc func(bsoncore.Value, bsoncore.Document) (bsoncore.Document, error)

var _ valueFixer = valueFixerFunc(nil)

func (vff valueFixerFunc) fixValue(val bsoncore.Value, dst bsoncore.Document) (bsoncore.Document, error) {
	return vff(val, dst)
}

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

		fixed, err = vf.fixValue(val, fixed)
		if err != nil {
			return nil, err
		}
	}

	fixed, _ = bsoncore.AppendDocumentEnd(fixed, idx)
	return fixed, nil
}

type documentValueFixer struct {
	key           string
	internalFixer compositeFixer
}

func newDocumentValueFixer(key string, cf compositeFixer) *documentValueFixer {
	return &documentValueFixer{
		key:           key,
		internalFixer: cf,
	}
}

func (dvf *documentValueFixer) fixValue(val bsoncore.Value, dst bsoncore.Document) (bsoncore.Document, error) {
	doc, ok := val.DocumentOK()
	if !ok {
		return nil, fmt.Errorf("expected value for key %s to be document, got %s", dvf.key, val.Type)
	}

	fixed, err := dvf.internalFixer.Fix(doc)
	if err != nil {
		return nil, err
	}
	dst = bsoncore.AppendDocumentElement(dst, dvf.key, fixed)
	return dst, nil
}
