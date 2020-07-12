package command

import (
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type Fixer interface {
	Fix(bsoncore.Document) (bsoncore.Document, error)
}

type fixerFunc func(bsoncore.Document) (bsoncore.Document, error)

func (f fixerFunc) Fix(doc bsoncore.Document) (bsoncore.Document, error) {
	return f(doc)
}

func noopFixer(doc bsoncore.Document) (bsoncore.Document, error) {
	return doc, nil
}

type valueFixer interface {
	fixValue(bsoncore.Value, string, bsoncore.Document) (bsoncore.Document, error)
}

type valueFixerFunc func(bsoncore.Value, string, bsoncore.Document) (bsoncore.Document, error)

func (vff valueFixerFunc) fixValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	return vff(val, key, dst)
}

type compositeFixer map[string]valueFixer

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

		fixed, err = vf.fixValue(val, key, fixed)
		if err != nil {
			return nil, err
		}
	}

	fixed, _ = bsoncore.AppendDocumentEnd(fixed, idx)
	return fixed, nil
}

type documentValueFixer struct {
	internalFixer Fixer
}

func newDocumentValueFixer(cf Fixer) *documentValueFixer {
	return &documentValueFixer{
		internalFixer: cf,
	}
}

func (dvf *documentValueFixer) fixValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	doc, ok := val.DocumentOK()
	if !ok {
		return nil, fmt.Errorf("expected value for key %s to be document, got %s", key, val.Type)
	}

	fixed, err := dvf.internalFixer.Fix(doc)
	if err != nil {
		return nil, err
	}
	dst = bsoncore.AppendDocumentElement(dst, key, fixed)
	return dst, nil
}

type arrayValueFixer struct {
	internalFixer valueFixer
}

func newArrayValueFixer(vf valueFixer) *arrayValueFixer {
	return &arrayValueFixer{
		internalFixer: vf,
	}
}

func (avf *arrayValueFixer) fixValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	arr, ok := val.ArrayOK()
	if !ok {
		return nil, fmt.Errorf("expected value for key %s to be array, got %s", key, val.Type)
	}

	values, err := arr.Values()
	if err != nil {
		return nil, err
	}

	var idx int32
	idx, dst = bsoncore.AppendArrayElementStart(dst, key)
	for idx, val := range values {
		dst, err = avf.internalFixer.fixValue(val, strconv.Itoa(idx), dst)
		if err != nil {
			return nil, err
		}
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, idx)
	return dst, nil
}
