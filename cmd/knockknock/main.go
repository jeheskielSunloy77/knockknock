package main

import (
	"os"
	"runtime/debug"

	"github.com/jeheskielSunloy77/knockknock/internal/cli"
)

var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			version = info.Main.Version
		}
	}
}

func main() {
	os.Exit(cli.Run(cli.Config{
		Args:    os.Args[1:],
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: version,
	}))
}
