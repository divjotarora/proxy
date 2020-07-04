package mongowire

import (
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type opQuery struct {
	reqID                int32
	flags                wiremessage.QueryFlag
	dbName               string
	collName             string
	numberToSkip         int32
	numberToReturn       int32
	query                bsoncore.Document
	returnFieldsSelector bsoncore.Document
}

var _ Message = (*opQuery)(nil)

func (q *opQuery) CommandDocument() bsoncore.Document {
	return q.query
}

func (q *opQuery) Encode() []byte {
	var buffer []byte
	idx, buffer := wiremessage.AppendHeaderStart(buffer, q.reqID, 0, wiremessage.OpQuery)
	buffer = wiremessage.AppendQueryFlags(buffer, q.flags)
	buffer = wiremessage.AppendQueryFullCollectionName(buffer, q.collName)
	buffer = wiremessage.AppendQueryNumberToSkip(buffer, q.numberToSkip)
	buffer = wiremessage.AppendQueryNumberToReturn(buffer, q.numberToReturn)
	buffer = append(buffer, q.query...)
	if len(q.returnFieldsSelector) != 0 {
		// returnFieldsSelector is optional
		buffer = append(buffer, q.returnFieldsSelector...)
	}
	buffer = bsoncore.UpdateLength(buffer, idx, int32(len(buffer[idx:])))
	return buffer
}

func (q *opQuery) RequestID() int32 {
	return q.reqID
}

// see https://github.com/mongodb/mongo-go-driver/blob/v1.3.4/x/mongo/driver/topology/server_test.go#L302-L337
func decodeQuery(reqID int32, wm []byte) (*opQuery, error) {
	var ok bool
	q := opQuery{
		reqID: reqID,
	}

	q.flags, wm, ok = wiremessage.ReadQueryFlags(wm)
	if !ok {
		return nil, errors.New("malformed query message: missing OP_QUERY flags")
	}

	var ns string
	ns, wm, ok = wiremessage.ReadQueryFullCollectionName(wm)
	if !ok {
		return nil, errors.New("malformed query message: full collection name")
	}
	if idx := strings.IndexByte(ns, '.'); idx != -1 {
		q.dbName = ns[:idx]
		q.collName = ns[idx+1:]
	}

	q.numberToSkip, wm, ok = wiremessage.ReadQueryNumberToSkip(wm)
	if !ok {
		return nil, errors.New("malformed query message: number to skip")
	}

	q.numberToReturn, wm, ok = wiremessage.ReadQueryNumberToReturn(wm)
	if !ok {
		return nil, errors.New("malformed query message: number to return")
	}

	q.query, wm, ok = wiremessage.ReadQueryQuery(wm)
	if !ok {
		return nil, errors.New("malformed query message: query document")
	}

	if len(wm) > 0 {
		q.returnFieldsSelector, _, ok = wiremessage.ReadQueryReturnFieldsSelector(wm)
		if !ok {
			return nil, errors.New("malformed query message: return fields selector")
		}
	}

	return &q, nil
}
