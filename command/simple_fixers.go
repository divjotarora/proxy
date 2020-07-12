package command

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// valueFixerFunc to add a prefix for $db values in requests.
var addDBPrefixValueFixer valueFixerFunc = func(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	db, ok := val.StringValueOK()
	if !ok {
		return nil, fmt.Errorf("expected $db value to be string, got %s", val.Type)
	}

	fixedDB := db
	if _, ok := noopDatabaseNames[db]; !ok {
		fixedDB = fmt.Sprintf("fixed%s", db)
	}
	dst = bsoncore.AppendStringElement(dst, key, fixedDB)
	return dst, nil
}

// valueFixerFunc to remove the database name prefix in responses.
var removeDBPrefixValueFixer valueFixerFunc = func(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
	db, ok := val.StringValueOK()
	if !ok {
		return nil, fmt.Errorf("expected $db value to be string, got %s", val.Type)
	}

	fixedDB := db
	if _, ok := noopDatabaseNames[db]; !ok {
		fixedDB = db[5:] // remove "fixed" prefix
	}
	dst = bsoncore.AppendStringElement(dst, key, fixedDB)
	return dst, nil
}

var writeErrorsFixer = newArrayValueFixer(newDocumentValueFixer(compositeFixer{
	"errmsg": valueFixerFunc(func(val bsoncore.Value, key string, dst bsoncore.Document) (bsoncore.Document, error) {
		errmsg, ok := val.StringValueOK()
		if !ok {
			return dst, fmt.Errorf("expected errmsg value to be of type string, got %s", val.Type)
		}

		fixedErrMsg := strings.ReplaceAll(errmsg, "fixed", "")
		dst = bsoncore.AppendStringElement(dst, key, fixedErrMsg)
		return dst, nil
	}),
}))
