package mongo

import "go.mongodb.org/mongo-driver/x/bsonx/bsoncore"

// IsMasterResponse TODO
var (
	maxBSONObjectSize            int32 = 16777216
	maxMessageSizeBytes          int32 = 48000000
	maxWriteBatchSize            int32 = 100000
	logicalSessionTimeoutMinutes int32 = 30
	minWireVersion               int32 = 0
	maxWireVersion               int32 = 8

	isMasterResponseDocument = bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendBooleanElement(nil, "ismaster", true),
		bsoncore.AppendInt32Element(nil, "maxBsonObjectSize", maxBSONObjectSize),
		bsoncore.AppendInt32Element(nil, "maxMessageSizeBytes", maxMessageSizeBytes),
		bsoncore.AppendInt32Element(nil, "maxWriteBatchSize", maxWriteBatchSize),
		bsoncore.AppendInt32Element(nil, "logicalSessionTimeoutMinutes", logicalSessionTimeoutMinutes),
		bsoncore.AppendInt32Element(nil, "minWireVersion", minWireVersion),
		bsoncore.AppendInt32Element(nil, "maxWireVersion", maxWireVersion),
	)
)

// HandshakeIsMasterResponse TODO
func HandshakeIsMasterResponse(requestID int32) Message {
	return &opReply{
		respTo:      requestID,
		numReturned: 1,
		documents:   []bsoncore.Document{isMasterResponseDocument},
	}
}

// HeartbeatIsMasterResponse TODO
func HeartbeatIsMasterResponse(requestID int32) Message {
	return &opMsg{
		respTo: requestID,
		sections: []opMsgSection{
			&opMsgSectionSingle{document: isMasterResponseDocument},
		},
	}
}
