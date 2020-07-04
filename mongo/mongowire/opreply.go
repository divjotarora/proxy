package mongowire

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type opReply struct {
	respTo   int32
	flags    wiremessage.ReplyFlag
	document bsoncore.Document
}

var _ Message = (*opReply)(nil)

func newOpReply(requestID int32, doc bsoncore.Document) *opReply {
	return &opReply{
		respTo:   requestID,
		document: doc,
	}
}

func (r *opReply) CommandDocument() bsoncore.Document {
	return r.document
}

func (r *opReply) Encode() []byte {
	return r.EncodeFixed(r.document)
}

func (r *opReply) EncodeFixed(fixedDocument bsoncore.Document) []byte {
	var buffer []byte
	idx, buffer := wiremessage.AppendHeaderStart(buffer, 0, r.respTo, wiremessage.OpReply)
	buffer = wiremessage.AppendReplyFlags(buffer, r.flags)
	buffer = wiremessage.AppendReplyCursorID(buffer, 0)
	buffer = wiremessage.AppendReplyStartingFrom(buffer, 0)
	buffer = wiremessage.AppendReplyNumberReturned(buffer, 1)
	buffer = append(buffer, fixedDocument...)
	buffer = bsoncore.UpdateLength(buffer, idx, int32(len(buffer[idx:])))
	return buffer
}

func (r *opReply) RequestID() int32 {
	return 0
}

// see https://github.com/mongodb/mongo-go-driver/blob/v1.3.4/x/mongo/driver/operation.go#L1101-L1162
func decodeReply(respTo int32, wm []byte) (*opReply, error) {
	var ok bool
	r := opReply{
		respTo: respTo,
	}

	r.flags, wm, ok = wiremessage.ReadReplyFlags(wm)
	if !ok {
		return nil, errors.New("malformed reply message: missing OP_REPLY flags")
	}

	_, wm, ok = wiremessage.ReadReplyCursorID(wm)
	if !ok {
		return nil, errors.New("malformed reply message: cursor id")
	}

	_, wm, ok = wiremessage.ReadReplyStartingFrom(wm)
	if !ok {
		return nil, errors.New("malformed reply message: starting from")
	}

	_, wm, ok = wiremessage.ReadReplyNumberReturned(wm)
	if !ok {
		return nil, errors.New("malformed reply message: number returned")
	}

	documents, _, ok := wiremessage.ReadReplyDocuments(wm)
	if !ok {
		return nil, errors.New("malformed reply message: could not read documents from reply")
	}
	if len(documents) != 1 {
		return nil, fmt.Errorf("malformed reply message: reply contains %d documents, but only 1 is supported", len(documents))
	}
	r.document = documents[0]

	return &r, nil
}
