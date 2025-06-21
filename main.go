package main

import (
	"log"

	"perf-runner/cli"
)

func main() {
	app := cli.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

