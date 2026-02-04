package main

import (
	"github.com/baighasan/kubecraft/internal/cli"
	_ "github.com/baighasan/kubecraft/internal/cli/server"
)

func main() {
	cli.Execute()
}
