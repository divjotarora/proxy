package main

import "github.com/divjotarora/proxy/proxy"

var (
	network  = "tcp"
	addresss = ":33000"
)

func main() {
	proxy := proxy.NewProxy(network, addresss)
	if err := proxy.Run(); err != nil {
		panic(err)
	}
}
