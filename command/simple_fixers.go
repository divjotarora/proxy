package command

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	// noopDatabaseNames contains names of databases that should be proxied without fixing.
	noopDatabaseNames = map[string]struct{}{
		"admin": {},
	}
)

// ValueFixerFunc to add the database name prefix in requests.
var addDBPrefixValueFixer ValueFixerFunc = func(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
	db, ok := val.StringValueOK()
	if !ok {
		return nil, fmt.Errorf("expected $db value to be string, got %s", val.Type)
	}

	fixedDB := db
	if _, ok := noopDatabaseNames[db]; !ok {
		fixedDB = fmt.Sprintf("fixed%s", db)
	}
	dst = bsoncore.AppendStringElement(dst, string(key), fixedDB)
	return dst, nil
}

// ValueFixerFunc to remove the database name prefix in responsnes.
var removeDBPrefixValueFixer ValueFixerFunc = func(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
	db, ok := val.StringValueOK()
	if !ok {
		return nil, fmt.Errorf("expected $db value to be string, got %s", val.Type)
	}

	fixedDB := db
	if _, ok := noopDatabaseNames[db]; !ok {
		fixedDB = db[5:] // remove "fixed" prefix
	}
	dst = bsoncore.AppendStringElement(dst, string(key), fixedDB)
	return dst, nil
}

// ValueFixer implementation to remove the database name prefix from messages in the writeErrors array in responses.
var writeErrorsValueFixer ValueFixer = newArrayValueFixer(DocumentFixer{
	"errmsg": ValueFixerFunc(func(val bsoncore.Value, key []byte, dst bsoncore.Document) (bsoncore.Document, error) {
		errmsg, ok := val.StringValueOK()
		if !ok {
			return dst, fmt.Errorf("expected errmsg value to be of type string, got %s", val.Type)
		}

		fixedErrMsg := strings.ReplaceAll(errmsg, "fixed", "")
		dst = bsoncore.AppendStringElement(dst, string(key), fixedErrMsg)
		return dst, nil
	}),
})
