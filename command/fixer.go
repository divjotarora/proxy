package command

import (
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// ValueFixer is implemented by types that can fix a single value in a document and write the fixed value out to the
// provided destination document.
type ValueFixer interface {
	fixValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error)
}

// ValueFixerFunc is a standalone function implementation of ValueFixer.
type ValueFixerFunc func(bsoncore.Value, string, bsoncore.Document) (bsoncore.Document, error)

func (vff ValueFixerFunc) fixValue(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	return vff(val, key, dst)
}

// DocumentFixer represents a set of ValueFixer instances, each mapped to a BSON key.
type DocumentFixer map[string]ValueFixer

// Fix iterates over the provided document to fix values using the registered ValueFixer instances and returns the
// fixed document.
func (df DocumentFixer) Fix(doc bsoncore.Document) (bsoncore.Document, error) {
	elems, err := doc.Elements()
	if err != nil {
		return nil, err
	}

	idx, fixed := bsoncore.AppendDocumentStart(nil)
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		vf, ok := df[key]
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

// documentValueFixer is the ValueFixer implementation for BSON subdocument values.
type documentValueFixer struct {
	internalFixer DocumentFixer
}

func newDocumentValueFixer(cf DocumentFixer) *documentValueFixer {
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

// arrayValueFixer is the ValueFixer implementation for BSON arrays.
type arrayValueFixer struct {
	internalFixer ValueFixer
}

func newArrayValueFixer(vf ValueFixer) *arrayValueFixer {
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
