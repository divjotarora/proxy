package mongowire

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// Message TODO
type Message interface {
	CommandDocument() bsoncore.Document
	Encode() []byte
	RequestID() int32
}

type FixableMessage interface {
	Message
	EncodeFixed(bsoncore.Document) []byte
}

// Decode TODO
func Decode(wm []byte) (Message, error) {
	wmLength := len(wm)
	length, reqID, respTo, opCode, wmBody, ok := wiremessage.ReadHeader(wm)
	if !ok || int(length) > wmLength {
		return nil, errors.New("malformed wire message: insufficient bytes")
	}

	switch opCode {
	case wiremessage.OpQuery:
		query, err := decodeQuery(reqID, wmBody)
		if err != nil {
			return nil, err
		}
		return query, nil
	case wiremessage.OpMsg:
		msg, err := decodeMsg(reqID, respTo, wmBody)
		if err != nil {
			return nil, err
		}
		return msg, nil
	case wiremessage.OpReply:
		reply, err := decodeReply(respTo, wmBody)
		if err != nil {
			return nil, err
		}
		return reply, nil
	default:
		return nil, fmt.Errorf("unrecognized opcode %d", opCode)
	}
}
