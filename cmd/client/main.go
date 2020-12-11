package main

import (
	"context"

	"github.com/zllovesuki/G14Manager/client"
)

func main() {
	configurator := client.NewInterface()

	configurator.Serve(context.Background())
}
