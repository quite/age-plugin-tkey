package main

import (
	_ "embed"
	"flag"
	"log"
	"os"
)

// nolint:typecheck // Avoid lint error when the embedding file is missing.
// Makefile copies the device app binary here ./app.bin
//
//go:embed app.bin
var appBinary []byte

var le = log.New(os.Stderr, "", 0)

func main() {
	var generateOnly bool
	var agePlugin string
	flag.StringVar(&agePlugin, "age-plugin", "", "For choosing state machine")
	flag.BoolVar(&generateOnly, "generate", false, "Generate secret key")
	flag.BoolVar(&generateOnly, "g", false, "Generate secret key")
	flag.Parse()

	if generateOnly {
		if !generate() {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if agePlugin != "" {
		switch agePlugin {
		case "recipient-v1":
			le.Printf("WIP\n")
			os.Exit(1)
		case "identity-v1":
			le.Printf("WIP\n")
			os.Exit(1)
		default:
			le.Printf("unknown state machine\n")
			os.Exit(1)
		}
	}

}

func generate() bool {
	t := tkey{}
	if err := t.connect(); err != nil {
		le.Printf("connect failed: %s\n", err)
		os.Exit(1)
	}

	t.disconnect()
	return true
}
