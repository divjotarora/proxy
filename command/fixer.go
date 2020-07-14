package command

import (
	"fmt"

	"github.com/divjotarora/proxy/bsonutil"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// ValueFixer is implemented by types that can fix a single value in a document and write the fixed value out to the
// provided destination document.
type ValueFixer interface {
	fixValue(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error)
}

// ValueFixerFunc is a standalone function implementation of ValueFixer.
type ValueFixerFunc func(bsoncore.Value, []byte, bsoncore.Document) (bsoncore.Document, error)

func (vff ValueFixerFunc) fixValue(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
	return vff(val, key, dst)
}

// DocumentFixer represents a set of ValueFixer instances, each mapped to a BSON key.
type DocumentFixer map[string]ValueFixer

var _ ValueFixer = DocumentFixer{}

// Fix iterates over the provided document to fix values using the registered ValueFixer instances and returns the
// fixed document.
func (df DocumentFixer) Fix(doc bsoncore.Document) (bsoncore.Document, error) {
	idx, fixed := bsoncore.AppendDocumentStart(nil)
	fixed, err := df.fixHelper(doc, fixed)
	if err != nil {
		return nil, err
	}
	fixed, _ = bsoncore.AppendDocumentEnd(fixed, idx)
	return fixed, nil
}

// fixValue implements ValueFixer.
func (df DocumentFixer) fixValue(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
	src, ok := val.DocumentOK()
	if !ok {
		return nil, fmt.Errorf("expected value to be document, got %s", val.Type)
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, string(key))
	dst, err := df.fixHelper(src, dst)
	if err != nil {
		return dst, err
	}
	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	return dst, nil
}

func (df DocumentFixer) fixHelper(src, dst bsoncore.Document) (bsoncore.Document, error) {
	iter, err := bsonutil.NewIterator(src)
	if err != nil {
		return nil, err
	}

	for iter.Next() {
		elem := iter.Element()
		val := iter.Value()
		key := elem.KeyBytes() // Keeping key as []byte and converting to string lazily when needed saves allocations.

		vf, ok := df[string(key)]
		if !ok {
			dst = bsoncore.AppendValueElement(dst, string(key), val)
			continue
		}

		dst, err = vf.fixValue(val, key, dst)
		if err != nil {
			return nil, err
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

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

func (avf *arrayValueFixer) fixValue(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
	arr, ok := val.ArrayOK()
	if !ok {
		return nil, fmt.Errorf("expected value for key %s to be array, got %s", key, val.Type)
	}

	iter, err := bsonutil.NewIterator(arr)
	if err != nil {
		return dst, err
	}

	var idx int32
	idx, dst = bsoncore.AppendArrayElementStart(dst, string(key))

	for iter.Next() {
		elem := iter.Element()
		val := iter.Value()

		// Use KeyBytes instead of Key to avoid an allocation.
		dst, err = avf.internalFixer.fixValue(val, elem.KeyBytes(), dst)
		if err != nil {
			return nil, err
		}
	}
	if err := iter.Err(); err != nil {
		return dst, err
	}

	dst, _ = bsoncore.AppendArrayEnd(dst, idx)
	return dst, nil
}
