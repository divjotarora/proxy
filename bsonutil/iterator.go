package bsonutil

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// Iterator TODO
type Iterator struct {
	src  []byte
	elem bsoncore.Element
	err  error
}

// NewIterator TODO
func NewIterator(src []byte) (*Iterator, error) {
	if len(src) < 5 {
		return nil, fmt.Errorf("document must be of length 5 or greater, but got length %d", len(src))
	}

	return &Iterator{src: src[4:]}, nil // remove leading length bytes
}

// HasNext TODO
func (i *Iterator) HasNext() bool {
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

// Element TODO
func (i *Iterator) Element() bsoncore.Element {
	return i.elem
}

// Err TODO
func (i *Iterator) Err() error {
	return i.err
}
