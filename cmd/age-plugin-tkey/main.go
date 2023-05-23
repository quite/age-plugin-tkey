package main

import (
	"flag"
	"log"
	"os"
)

var le = log.New(os.Stderr, "", 0)

func main() {
	var agePlugin string
	flag.StringVar(&agePlugin, "age-plugin", "", "For choosing state machine")
	flag.Parse()

	if agePlugin == "" {
		le.Printf("Please pass --age-plugin=STATE_MACHINE\n")
		os.Exit(2)
	}

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
