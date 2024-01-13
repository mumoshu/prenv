package main

import (
	"os"

	"github.com/mumoshu/prenv/cmd"
)

func main() {
	if err := cmd.Main(); err != nil {
		os.Exit(1)
	}
}
