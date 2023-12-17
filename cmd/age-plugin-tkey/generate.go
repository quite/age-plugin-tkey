package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/quite/age-plugin-tkey/internal/identity"
	"github.com/quite/tkeyx25519"
	"golang.org/x/term"
)

func generate(out *os.File, requireTouch bool) bool {
	// Generate a privkey on the TKey and get hold of the pubkey

	var userSecret [tkeyx25519.UserSecretSize]byte
	if _, err := rand.Read(userSecret[:]); err != nil {
		le.Printf("rand.Read failed: %s\n", err)
		return false
	}

	udi, id, err := identity.NewIdentity(userSecret[:], requireTouch)
	if err != nil {
		le.Printf("NewIdentity failed: %s\n", err)
		return false
	}

	idStr, err := id.EncodeIdentity(pluginName)
	if err != nil {
		le.Printf("%s\n", err)
		return false
	}

	recipient, err := id.EncodeRecipient()
	if err != nil {
		le.Printf("encodeRecipient failed: %s\n", err)
		return false
	}

	if udi == nil {
		le.Printf("Could not get TKey serial number (UDI) because device app was already loaded. Re-plug TKey and generate again if you need it.\n")
	}
	if !term.IsTerminal(int(out.Fd())) {
		le.Printf("recipient: %s\n", recipient)
	}

	fmt.Fprintf(out, "# created: %s\n", time.Now().UTC().Format(time.RFC3339))
	if udi != nil {
		fmt.Fprintf(out, "# TKey serial number (UDI): %s\n", udi.String())
	}
	fmt.Fprintf(out, "# recipient: %s\n", recipient)
	fmt.Fprintf(out, "# touch required: %t\n", requireTouch)
	fmt.Fprintf(out, "%s\n", idStr)

	return true
}
