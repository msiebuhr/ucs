package main

import (
	"context"

	"gitlab.com/msiebuhr/ucs"

	"github.com/namsral/flag"
)

func main() {
	var address string
	flag.StringVar(&address, "address", ":8126", "Address and port to listen on")

	flag.Parse()

	server := ucs.NewServer()

	server.Listen(context.Background(), "tcp", address)
}
