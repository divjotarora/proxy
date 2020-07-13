package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/divjotarora/proxy/command"
	"github.com/divjotarora/proxy/connection"
	conn "github.com/divjotarora/proxy/connection"
	"github.com/divjotarora/proxy/mongo"
	"github.com/divjotarora/proxy/mongo/mongowire"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	emptyFixerSet = command.FixerSet{}
)

// Proxy represents a network proxy that sits between a client and a MongoDB server.
type Proxy struct {
	network   string
	address   string
	client    *mongo.Client
	parser    *command.Parser
	wg        sync.WaitGroup
	cursorMap map[int64]string // cursor ID -> originating command name
}

// NewProxy creates a new Proxy instance.
func NewProxy(network, address string, clientOpts *options.ClientOptions) (*Proxy, error) {
	client, err := mongo.NewClient(context.TODO(), clientOpts)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		network:   network,
		address:   address,
		client:    client,
		parser:    command.NewParser(),
		cursorMap: make(map[int64]string),
	}
	return p, nil
}

// Run starts the proxy. This method blocks indefinitely listening for new connections and starts a goroutine to
// handle messages for each accepted connection.
func (p *Proxy) Run() error {
	listener, err := net.Listen(p.network, p.address)
	if err != nil {
		return fmt.Errorf("Listen error: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	log.Println("waiting for new connections")
	for {
		nc, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Accept error: %w", err)
		}
		log.Printf("accepted connection from address %s\n", nc.RemoteAddr())

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			defer func() {
				_ = nc.Close()
			}()

			userConn, err := conn.NewConn(nc)
			if err != nil {
				log.Printf("error establishing user connection: %v\n", err)
				return
			}
			if err := p.handleConnection(userConn); err != nil {
				if errors.Is(err, connection.ErrClientHungUp) {
					log.Println("connection closed by client")
					return
				}

				log.Printf("handleConnection error: %v", err)
			}
		}()
	}
}

func (p *Proxy) handleConnection(conn *conn.Connection) error {
	for {
		if err := p.handleRequest(conn); err != nil {
			return err
		}
	}
}

func (p *Proxy) handleRequest(conn *conn.Connection) error {
	msgBytes, err := conn.ReadWireMessage(nil)
	if err != nil {
		return err
	}

	msg, err := mongowire.Decode(msgBytes)
	if err != nil {
		return err
	}

	cmd := msg.CommandDocument()

	switch cmdName := cmd.Index(0).Key(); cmdName {
	case "isMaster", "ismaster":
		heartbeatResponse := mongowire.HeartbeatIsMasterResponse(msg.RequestID())
		return conn.WriteWireMessage(heartbeatResponse.Encode())
	default:
		return p.handleProxiedRequest(msg, cmdName, conn)
	}
}

func (p *Proxy) handleProxiedRequest(requestMsg mongowire.Message, cmdName string, conn *connection.Connection) error {
	fixerSet, err := p.getFixerSet(cmdName, requestMsg.CommandDocument())
	if err != nil {
		return err
	}

	// Get a wire message for the fixed request.
	fixedRequest, err := fixerSet.FixRequest(requestMsg.CommandDocument())
	if err != nil {
		return err
	}
	encodedRequest := requestMsg.EncodeFixed(fixedRequest)

	// Send the fixed request to the server and get a response.
	responseBytes, err := p.client.RoundTrip(context.TODO(), encodedRequest)
	if err != nil {
		return err
	}
	responseMsg, err := mongowire.Decode(responseBytes)
	if err != nil {
		return err
	}

	cursorID := getCursorID(responseMsg.CommandDocument())
	if cmdName == "getMore" {
		// If this is the last getMore on the cursor, remove the cursor from the map.
		if cursorID == 0 {
			delete(p.cursorMap, fixedRequest.Index(0).Value().Int64())
		}
	} else if cursorID != 0 {
		// If the response has a cursor ID, this is a cursor-creating command. Track the ID and command name so we
		// know how to fix future getMore responses.
		p.cursorMap[cursorID] = cmdName
	}

	// Get a wire message for the fixed response and send that back to the client.
	fixedResponse, err := fixerSet.FixResponse(responseMsg.CommandDocument())
	if err != nil {
		return err
	}
	encodedResponse := responseMsg.EncodeFixed(fixedResponse)
	return conn.WriteWireMessage(encodedResponse)
}

func (p *Proxy) getFixerSet(cmdName string, doc bsoncore.Document) (command.FixerSet, error) {
	// For getMore requests, get the fixer set for the originating command.
	if cmdName == "getMore" {
		cursorIDVal := doc.Index(0).Value()
		cursorID, ok := cursorIDVal.Int64OK()
		if !ok {
			return emptyFixerSet, fmt.Errorf("expected getMore value to be int64, got %s", cursorIDVal.Type)
		}

		originalCmdName, ok := p.cursorMap[cursorID]
		if !ok {
			return emptyFixerSet, fmt.Errorf("dangling cursor ID %v", cursorID)
		}
		cmdName = originalCmdName
	}

	return p.parser.Parse(cmdName), nil
}

func getCursorID(doc bsoncore.Document) int64 {
	cursorIDVal, err := doc.LookupErr("cursor", "id")
	if err != nil {
		return 0
	}

	cursorID, ok := cursorIDVal.Int64OK()
	if !ok {
		return 0
	}
	return cursorID
}
