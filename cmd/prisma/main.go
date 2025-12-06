package main

import (
	"os"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
