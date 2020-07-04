package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/divjotarora/proxy/command"
	"github.com/divjotarora/proxy/conn"
	"github.com/divjotarora/proxy/mongo"
	"github.com/divjotarora/proxy/mongo/mongowire"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Proxy TODO
type Proxy struct {
	network string
	address string
	client  *mongo.Client
	parser  *command.Parser
	wg      sync.WaitGroup
}

// NewProxy TODO
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

// Run TODO
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
				log.Printf("handleConnection error: %v", err)
			}
		}()
	}
}

func (p *Proxy) handleConnection(conn *conn.Conn) error {
	for {
		if err := p.handleRequest(conn); err != nil {
			return err
		}
	}
}

func (p *Proxy) handleRequest(conn *conn.Conn) error {
	msg, err := conn.ReadWireMessage(nil)
	if err != nil {
		return fmt.Errorf("error reading user message: %w", err)
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
