package mongowire

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type opMsgSection struct {
	sectionType wiremessage.SectionType
	document    bsoncore.Document
	identifier  string
	sequence    []bsoncore.Document
}

type opMsg struct {
	reqID    int32
	respTo   int32
	doc      bsoncore.Document
	flags    wiremessage.MsgFlag
	sections []*opMsgSection
}

var _ Message = (*opMsg)(nil)
var _ FixableMessage = (*opMsg)(nil)

func newOpMsgResponse(requestID int32, doc bsoncore.Document) *opMsg {
	section := &opMsgSection{
		sectionType: wiremessage.SingleDocument,
		document:    doc,
	}
	return &opMsg{
		respTo:   requestID,
		doc:      doc,
		sections: []*opMsgSection{section},
	}
}

func (m *opMsg) CommandDocument() bsoncore.Document {
	return m.doc
}

func (m *opMsg) Encode() []byte {
	return m.EncodeFixed(m.doc)
}

func (m *opMsg) EncodeFixed(fixedCmd bsoncore.Document) []byte {
	var buffer []byte
	idx, buffer := wiremessage.AppendHeaderStart(buffer, m.reqID, m.respTo, wiremessage.OpMsg)
	buffer = wiremessage.AppendMsgFlags(buffer, m.flags)
	for _, section := range m.sections {
		buffer = wiremessage.AppendMsgSectionType(buffer, section.sectionType)

		switch section.sectionType {
		case wiremessage.SingleDocument:
			buffer = append(buffer, fixedCmd...)
		case wiremessage.DocumentSequence:
			length := int32(len(section.identifier) + 5)
			for _, msg := range section.sequence {
				length += int32(len(msg))
			}

			buffer = appendi32(buffer, length)
			buffer = appendCString(buffer, section.identifier)
			for _, msg := range section.sequence {
				buffer = append(buffer, msg...)
			}
		}
	}

	buffer = bsoncore.UpdateLength(buffer, idx, int32(len(buffer[idx:])))
	return buffer
}

func (m *opMsg) RequestID() int32 {
	return m.reqID
}

// see https://github.com/mongodb/mongo-go-driver/blob/v1.3.4/x/mongo/driver/operation.go#L1191-L1220
func decodeMsg(reqID, respTo int32, wm []byte) (*opMsg, error) {
	var ok bool
	m := opMsg{
		reqID:  reqID,
		respTo: respTo,
	}

	m.flags, wm, ok = wiremessage.ReadMsgFlags(wm)
	if !ok {
		return nil, errors.New("malformed wire message: missing OP_MSG flags")
	}

	for len(wm) > 0 {
		var section opMsgSection
		section.sectionType, wm, ok = wiremessage.ReadMsgSectionType(wm)
		if !ok {
			return nil, errors.New("malformed wire message: insufficient bytes to read section type")
		}

		switch section.sectionType {
		case wiremessage.SingleDocument:
			section.document, wm, ok = wiremessage.ReadMsgSectionSingleDocument(wm)
			if !ok {
				return nil, errors.New("malformed wire message: insufficient bytes to read single document")
			}
			m.doc = section.document
		case wiremessage.DocumentSequence:
			section.identifier, section.sequence, wm, ok = wiremessage.ReadMsgSectionDocumentSequence(wm)
			if !ok {
				return nil, errors.New("malformed wire message: insufficient bytes to read document sequence")
			}
		default:
			return nil, fmt.Errorf("malformed wire message: unknown section type %v", section.sectionType)
		}

		m.sections = append(m.sections, &section)
	}

	return &m, nil
}

func appendi32(dst []byte, i32 int32) []byte {
	return append(dst, byte(i32), byte(i32>>8), byte(i32>>16), byte(i32>>24))
}

func appendCString(b []byte, str string) []byte {
	b = append(b, str...)
	return append(b, 0x00)
}
