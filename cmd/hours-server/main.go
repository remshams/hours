package main

import (
	"os"

	serverpkg "github.com/dhth/hours/internal/server"
)

func main() {
	if err := serverpkg.Execute(); err != nil {
		os.Exit(1)
	}
}
