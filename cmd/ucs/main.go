package main

import (
	"context"

	"gitlab.com/msiebuhr/ucs"
)

func main() {
	server := ucs.NewServer()

	server.Listen(context.Background())
}
