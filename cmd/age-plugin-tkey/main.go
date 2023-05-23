package main

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"filippo.io/age/plugin"
	"golang.org/x/crypto/blake2s"
)

const (
	pluginName = "tkey"
	// Max length is 78 bytes:
	fixedDomain = "tillitis.se/tkey"
)

// nolint:typecheck // Avoid lint error when the embedding file is
// missing. Makefile copies the device app binary here ./app.bin
//
//go:embed app.bin
var appBinary []byte

var le = log.New(os.Stderr, "", 0)

func main() {
	var generateOnly, requireTouch bool
	var agePlugin string
	flag.StringVar(&agePlugin, "age-plugin", "", "For choosing state machine")
	descGenerate := "Generate an identity backed by TKey"
	flag.BoolVar(&generateOnly, "generate", false, descGenerate)
	flag.BoolVar(&generateOnly, "g", false, descGenerate)
	flag.BoolVar(&requireTouch, "touch", false, "Require physical touch of TKey upon use of identity")
	flag.Parse()

	if generateOnly {
		if !generate(requireTouch) {
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

func generate(requireTouch bool) bool {
	t := tkey{}
	if err := t.connect(); err != nil {
		le.Printf("connect failed: %s\n", err)
		return false
	}
	defer t.disconnect()

	// Generate a privkey on the TKey and get hold of the pubkey

	var domain [78]byte
	copy(domain[:], fixedDomain)

	var userSecret [16]byte
	if _, err := rand.Read(userSecret[:]); err != nil {
		le.Printf("rand.Read failed: %s\n", err)
		return false
	}

	// TODO turning off random userSecret above, it looks like the
	// pubkey doesn't change depending on touchRequired when run on
	// hw, but it does in qemu!?
	pubBytes, err := t.x25519.GetPubKey(domain, userSecret, requireTouch)
	if err != nil {
		le.Printf("GetPubKey failed: %s\n", err)
		return false
	}

	pub, err := ecdh.X25519().NewPublicKey(pubBytes)
	if err != nil {
		le.Printf("NewPublicKey failed: %s\n", err)
		return false
	}
	recipient, err := plugin.EncodeX25519Recipient(pub)
	if err != nil {
		le.Printf("EncodeX25519Recipient failed: %s\n", err)
		return false
	}

	// Now generate an identity using the non-fixed params including a
	// short hash of the pubkey for later check.

	var params bytes.Buffer
	params.Write(userSecret[:])
	if requireTouch {
		params.WriteByte(1)
	} else {
		params.WriteByte(0)
	}
	pubHash := blake2s.Sum256(pubBytes)
	params.Write(pubHash[:2])

	identity := plugin.EncodeIdentity(pluginName, params.Bytes())
	if identity == "" {
		le.Printf("EncodeIdentity returned empty string\n")
		return false
	}

	le.Printf("# recipient: %s\n", recipient)
	le.Printf("# touch required: %t\n", requireTouch)
	fmt.Printf("%s\n", identity)

	return true
}
