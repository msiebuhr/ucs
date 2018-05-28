package main

import (
	"context"

	"gitlab.com/msiebuhr/ucs"
)

func main() {
	ucs.Listen(context.Background())
}
