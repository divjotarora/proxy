package mongowire

import (
	"errors"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type opReply struct {
	respTo       int32
	flags        wiremessage.ReplyFlag
	cursorID     int64
	startingFrom int32
	numReturned  int32
	documents    []bsoncore.Document
}

var _ Message = (*opReply)(nil)

func (r *opReply) CommandDocument() bsoncore.Document {
	return r.documents[0]
}

func (r *opReply) Encode() []byte {
	var buffer []byte
	idx, buffer := wiremessage.AppendHeaderStart(buffer, 0, r.respTo, wiremessage.OpReply)
	buffer = wiremessage.AppendReplyFlags(buffer, r.flags)
	buffer = wiremessage.AppendReplyCursorID(buffer, r.cursorID)
	buffer = wiremessage.AppendReplyStartingFrom(buffer, r.startingFrom)
	buffer = wiremessage.AppendReplyNumberReturned(buffer, r.numReturned)
	for _, doc := range r.documents {
		buffer = append(buffer, doc...)
	}
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

	r.cursorID, wm, ok = wiremessage.ReadReplyCursorID(wm)
	if !ok {
		return nil, errors.New("malformed reply message: cursor id")
	}

	r.startingFrom, wm, ok = wiremessage.ReadReplyStartingFrom(wm)
	if !ok {
		return nil, errors.New("malformed reply message: starting from")
	}

	r.numReturned, wm, ok = wiremessage.ReadReplyNumberReturned(wm)
	if !ok {
		return nil, errors.New("malformed reply message: number returned")
	}

	r.documents, _, ok = wiremessage.ReadReplyDocuments(wm)
	if !ok {
		return nil, errors.New("malformed reply message: could not read documents from reply")
	}

	return &r, nil
}
