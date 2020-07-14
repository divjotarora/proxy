package bsonutil

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	// MinDocumentSize is the minimum size of a BSON document or array.
	MinDocumentSize = 5

	// ErrDocumentTooSmall is returned if a document's length is below MinDocumentSize.
	ErrDocumentTooSmall = fmt.Errorf("document must be of length at least %d", MinDocumentSize)
)

// Iterator represents a lazy iterator over a BSON document or array.
type Iterator struct {
	src  []byte
	elem bsoncore.Element
	err  error
}

// NewIterator creates an iterator for the provided source BSON document or array. This function returns
// ErrDocumentTooSmall if the document size is invalid.
func NewIterator(src []byte) (*Iterator, error) {
	if len(src) < MinDocumentSize {
		return nil, ErrDocumentTooSmall
	}

	return &Iterator{src: src[4:]}, nil
}

// Next returns true if there is another element to read and false if the document has been exhausted or the the
// iterator encountered an error while reading. Iteration errors can be retrieved via the Err function.
func (i *Iterator) Next() bool {
	// If there is one byte left, it's the terminating 0 byte of the document, which should be ignored, so iteration
	// is complete.
	if len(i.src) <= 1 {
		return false
	}

	var ok bool
	i.elem, i.src, ok = bsoncore.ReadElement(i.src)
	if !ok {
		i.err = errors.New("ReadElement failed")
		return false
	}
	if i.err = i.elem.Validate(); i.err != nil {
		return false
	}

	return true
}

// Element returns the last element read by the iterator. The element returned is only valid until the subsequent Next
// call.
func (i *Iterator) Element() bsoncore.Element {
	return i.elem
}

// Value returns the last value read by the iterator. The value returned is only valid until the subsequent Next call.
func (i *Iterator) Value() bsoncore.Value {
	return i.elem.Value()
}

// Err returns the last iteration error seen, or nil if there were no errors.
func (i *Iterator) Err() error {
	return i.err
}
