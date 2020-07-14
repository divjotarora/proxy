package bsonutil

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// ValueToByteSlice converts a BSON string value into a []byte. This is equivalent to calling
// bsoncore.Value.StringValueOK(), except it returns the value as []byte rather than string to minimize allocations.
func ValueToByteSlice(val bsoncore.Value) ([]byte, bool) {
	if val.Type != bsontype.String {
		return nil, false
	}

	strlen, rem, ok := bsoncore.ReadLength(val.Data)
	if !ok {
		return nil, false
	}
	if len(val.Data[4:]) < int(strlen) {
		return nil, false
	}

	return rem[:strlen-1], true
}
