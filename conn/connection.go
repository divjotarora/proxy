package conn

import (
	"fmt"
	"io"
	"net"

	"github.com/divjotarora/proxy/mongo"
)

// Conn TODO
type Conn struct {
	net.Conn
}

// NewConn TODO
func NewConn(nc net.Conn) (*Conn, error) {
	c := &Conn{
		nc,
	}

	if err := c.handshake(); err != nil {
		return nil, err
	}
	return c, nil
}

// ReadWireMessage TODO
func (c *Conn) ReadWireMessage(buf []byte) (mongo.Message, error) {
	var sizeBuf [4]byte

	_, err := io.ReadFull(c, sizeBuf[:])
	if err != nil {
		return nil, err
	}

	// read the length as an int32
	size := (int32(sizeBuf[0])) | (int32(sizeBuf[1]) << 8) | (int32(sizeBuf[2]) << 16) | (int32(sizeBuf[3]) << 24)
	if int(size) > cap(buf) {
		buf = make([]byte, 0, size)
	}

	buffer := buf[:size]
	copy(buffer, sizeBuf[:])

	_, err = io.ReadFull(c, buffer[4:])
	if err != nil {
		return nil, err
	}

	msg, err := mongo.Decode(buffer)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// WriteWireMessage TODO
func (c *Conn) WriteWireMessage(buf []byte) error {
	_, err := c.Write(buf)
	return err
}

// Handshake TODO
func (c *Conn) handshake() error {
	for {
		msg, err := c.ReadWireMessage(nil)
		if err != nil {
			return err
		}

		cmd := msg.CommandDocument()
		cmdName := cmd.Index(0).Key()

		switch cmdName {
		case "isMaster", "ismaster":
			response := mongo.HandshakeIsMasterResponse(msg.RequestID())
			return c.WriteWireMessage(response.Encode())
		default:
			return fmt.Errorf("unknown handshake command %s", cmdName)
		}
	}
}
