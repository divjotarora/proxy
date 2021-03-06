package mongo

import (
	"context"
	"reflect"
	"unsafe"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

// Client represents a direct connection to a MongoDB server. This is a long-lived type and is safe for concurrent use.
type Client struct {
	client *mongo.Client
	server driver.Server
}

// NewClient creates a new Client instance.
func NewClient(ctx context.Context, opts *options.ClientOptions) (*Client, error) {
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	topo := extractTopology(client)
	server, err := topo.SelectServer(ctx, description.ReadPrefSelector(readpref.Primary()))
	if err != nil {
		// Use context.Background to ensure client is properly disconnected even if ctx has expired.
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	c := &Client{
		client: client,
		server: server,
	}
	return c, nil
}

// Disconnect closes open connections to the MongoDB server and cleans up any remaining resources.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// RoundTrip sends a wire message to the underlying MongoDB server and returns the server's response.
func (c *Client) RoundTrip(ctx context.Context, msg []byte) ([]byte, error) {
	conn, err := c.server.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.WriteWireMessage(ctx, msg); err != nil {
		return nil, err
	}

	return conn.ReadWireMessage(ctx, nil)
}

func extractTopology(c *mongo.Client) *topology.Topology {
	e := reflect.ValueOf(c).Elem()
	d := e.FieldByName("deployment")
	d = reflect.NewAt(d.Type(), unsafe.Pointer(d.UnsafeAddr())).Elem() // #nosec G103
	return d.Interface().(*topology.Topology)
}
