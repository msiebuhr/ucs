package main

import (
	"context"

	"github.com/msiebuhr/unity-cache-server"
)

func main() {
	ucs.Listen(context.Background())
}
