package main

import (
	"os"

	"knockknock/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(cli.Config{
		Args:    os.Args[1:],
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: version,
	}))
}
