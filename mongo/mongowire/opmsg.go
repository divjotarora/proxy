package mongowire

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type opMsg struct {
	reqID    int32
	respTo   int32
	dbName   string
	flags    wiremessage.MsgFlag
	sections []opMsgSection
}

var _ Message = (*opMsg)(nil)

type opMsgSection interface {
	append([]byte) []byte
}

type opMsgSectionSingle struct {
	document bsoncore.Document
}

func (o *opMsgSectionSingle) append(buffer []byte) []byte {
	buffer = wiremessage.AppendMsgSectionType(buffer, wiremessage.SingleDocument)
	return append(buffer, o.document...)
}

type opMsgSectionSequence struct {
	identifier string
	msgs       []bsoncore.Document
}

func (o *opMsgSectionSequence) append(buffer []byte) []byte {
	buffer = wiremessage.AppendMsgSectionType(buffer, wiremessage.DocumentSequence)

	length := int32(len(o.identifier) + 5)
	for _, msg := range o.msgs {
		length += int32(len(msg))
	}

	buffer = appendi32(buffer, length)
	buffer = appendCString(buffer, o.identifier)
	for _, msg := range o.msgs {
		buffer = append(buffer, msg...)
	}

	return buffer
}

func (m *opMsg) CommandDocument() bsoncore.Document {
	for _, section := range m.sections {
		if single, ok := section.(*opMsgSectionSingle); ok {
			return single.document
		}
	}
	return nil
}

func (m *opMsg) Database() string {
	return m.dbName
}

func (m *opMsg) Encode() []byte {
	var buffer []byte
	idx, buffer := wiremessage.AppendHeaderStart(buffer, m.reqID, m.respTo, wiremessage.OpMsg)
	buffer = wiremessage.AppendMsgFlags(buffer, m.flags)
	for _, section := range m.sections {
		buffer = section.append(buffer)
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
		var stype wiremessage.SectionType
		stype, wm, ok = wiremessage.ReadMsgSectionType(wm)
		if !ok {
			return nil, errors.New("malformed wire message: insufficient bytes to read section type")
		}

		switch stype {
		case wiremessage.SingleDocument:
			s := opMsgSectionSingle{}
			s.document, wm, ok = wiremessage.ReadMsgSectionSingleDocument(wm)
			if !ok {
				return nil, errors.New("malformed wire message: insufficient bytes to read single document")
			}
			m.sections = append(m.sections, &s)

			dbVal, err := s.document.LookupErr("$db")
			if err == nil {
				// Messages from the server to the original client might not have a $db field.
				m.dbName = dbVal.StringValue()
			}
		case wiremessage.DocumentSequence:
			s := opMsgSectionSequence{}
			s.identifier, s.msgs, wm, ok = wiremessage.ReadMsgSectionDocumentSequence(wm)
			if !ok {
				return nil, errors.New("malformed wire message: insufficient bytes to read document sequence")
			}
			m.sections = append(m.sections, &s)
		default:
			return nil, fmt.Errorf("malformed wire message: unknown section type %v", stype)
		}
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
