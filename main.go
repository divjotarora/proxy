package main

import (
	"github.com/divjotarora/proxy/proxy"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	network  = "tcp"
	addresss = ":33000"
	mongoURI = "mongodb://localhost:27017"
)

func main() {
	clientOpts := options.Client().ApplyURI(mongoURI)
	proxy, err := proxy.NewProxy(network, addresss, clientOpts)
	if err != nil {
		panic(err)
	}

	if err := proxy.Run(); err != nil {
		panic(err)
	}
}
