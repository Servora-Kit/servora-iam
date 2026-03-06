package main

import (
	"os"

	"github.com/Servora-Kit/servora/cmd/svr/internal/root"
)

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
