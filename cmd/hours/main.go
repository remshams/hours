package main

import (
	"os"

	"github.com/dhth/hours/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
