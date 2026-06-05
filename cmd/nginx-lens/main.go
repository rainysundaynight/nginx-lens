package main

import (
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
