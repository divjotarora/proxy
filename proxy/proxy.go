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
)

// Proxy represents a network proxy that sits between a client and a MongoDB server.
type Proxy struct {
	network string
	address string
	client  *mongo.Client
	parser  *command.Parser
	wg      sync.WaitGroup
}

// NewProxy creates a new Proxy instance.
func NewProxy(network, address string, clientOpts *options.ClientOptions) (*Proxy, error) {
	client, err := mongo.NewClient(context.TODO(), clientOpts)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		network: network,
		address: address,
		client:  client,
		parser:  command.NewParser(),
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
	var responseMsg mongowire.Message

	switch cmdName := cmd.Index(0).Key(); cmdName {
	case "isMaster", "ismaster":
		responseMsg = mongowire.HeartbeatIsMasterResponse(msg.RequestID())
	default:
		responseMsg, err = p.handleProxiedRequest(msg, cmdName)
	}
	if err != nil {
		return fmt.Errorf("error handling request: %w", err)
	}

	return conn.WriteWireMessage(responseMsg.Encode())
}

func (p *Proxy) handleProxiedRequest(msg mongowire.Message, cmdName string) (mongowire.Message, error) {
	fixer := p.parser.Parse(cmdName)
	fixedCmd, err := fixer.Fix(msg.CommandDocument())
	if err != nil {
		return nil, err
	}

	fixableMsg, ok := msg.(mongowire.FixableMessage)
	if !ok {
		return nil, fmt.Errorf("expected message of type %T to be a FixableMessage", msg)
	}

	encoded := fixableMsg.EncodeFixed(fixedCmd)
	responseBytes, err := p.client.RoundTrip(context.TODO(), encoded)
	if err != nil {
		return nil, err
	}

	return mongowire.Decode(responseBytes)
}
