package conn

import (
	"fmt"
	"io"
	"net"

	"github.com/divjotarora/proxy/mongo/mongowire"
)

// Conn represents a network connection between a client and the proxy.
type Conn struct {
	net.Conn
}

// NewConn creates a new Conn instance wrapping the underlying net.Conn. This function performs all handshake commands
// necessary to initialize the connection.
func NewConn(nc net.Conn) (*Conn, error) {
	c := &Conn{
		nc,
	}

	if err := c.handshake(); err != nil {
		return nil, err
	}
	return c, nil
}

// ReadWireMessage reads the next wire message from the client.
func (c *Conn) ReadWireMessage(buf []byte) ([]byte, error) {
	var sizeBuf [4]byte

	_, err := io.ReadFull(c, sizeBuf[:])
	if err != nil {
		return nil, err
	}

	// Read the length as an int32
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

	return buffer, nil
}

// WriteWireMessage writes the given wire message to the client.
func (c *Conn) WriteWireMessage(buf []byte) error {
	_, err := c.Write(buf)
	return err
}

func (c *Conn) handshake() error {
	for {
		msgBytes, err := c.ReadWireMessage(nil)
		if err != nil {
			return err
		}
		msg, err := mongowire.Decode(msgBytes)
		if err != nil {
			return err
		}

		cmd := msg.CommandDocument()
		cmdName := cmd.Index(0).Key()

		switch cmdName {
		case "isMaster", "ismaster":
			response := mongowire.HandshakeIsMasterResponse(msg.RequestID())
			return c.WriteWireMessage(response.Encode())
		default:
			return fmt.Errorf("unknown handshake command %s", cmdName)
		}
	}
}
