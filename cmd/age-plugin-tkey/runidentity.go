package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/quite/age-plugin-tkey/internal/identity"
	"golang.org/x/crypto/curve25519"
)

type recipient struct {
	fileIndex      string
	pubKey         []byte
	wrappedFileKey []byte
}

func runIdentity() error {
	identities := []*identity.Identity{}
	recipients := []*recipient{}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			return fmt.Errorf("stanza has less than 2 parts")
		}
		tag, typ, args := parts[0], parts[1], parts[2:]

		if tag != "->" {
			return fmt.Errorf("stanza must begin with '->'")
		}

		switch typ {
		case "add-identity":
			if len(args) < 1 {
				return fmt.Errorf("add-identity must have 1 arg")
			}

			name, idBytes, err := plugin.ParseIdentity(args[0])
			if err != nil {
				return fmt.Errorf("ParseIdentity failed: %w", err)
			}
			if name != pluginName {
				le.Printf("identity skipped: plugin name is %s, expected %s\n", name, pluginName)
				continue
			}

			id, err := identity.NewIdentityFromBytes(idBytes)
			if err != nil {
				// TODO should some error in there make us error out?
				le.Printf("identity skipped: NewIdentityFromBytes failed: %s\n", err)
				continue
			}

			identities = append(identities, id)

		case "recipient-stanza":
			if len(args) < 3 {
				return fmt.Errorf("recipient-stanza must have 3 args")
			}

			fileIndex, recipientType, recipientPubKeyStr := args[0], args[1], args[2]
			if recipientType != "X25519" {
				le.Printf("recipient skipped: type is %s, expected X25519\n", recipientType)
				continue
			}

			recipientPubKey, err := DecodeString(recipientPubKeyStr)
			if err != nil {
				return fmt.Errorf("decode recipientPubKey failed: %w", err)
			}
			if len(recipientPubKey) != curve25519.PointSize {
				return fmt.Errorf("recipientPubKey has wrong length")
			}

			var wrappedFileKeyStr string
			for scanner.Scan() {
				line = scanner.Text()
				wrappedFileKeyStr += line
				if len(line) < 64 {
					break
				}
			}
			wrappedFileKey, err := DecodeString(wrappedFileKeyStr)
			if err != nil {
				return fmt.Errorf("decode wrappedFileKey failed: %w", err)
			}

			recipients = append(recipients, &recipient{
				fileIndex:      fileIndex,
				pubKey:         recipientPubKey,
				wrappedFileKey: wrappedFileKey,
			})
		}

		if typ == "done" {
			break
		}
	}

	if len(identities) == 0 {
		return fmt.Errorf("no identities specified")
	}

	for _, rcpt := range recipients {
		for _, id := range identities {
			fileKey, err := id.Unwrap(rcpt.pubKey, rcpt.wrappedFileKey)
			if err != nil {
				if errors.Is(err, age.ErrIncorrectIdentity) {
					continue
				}
				return err
			}

			fmt.Printf("-> file-key %s\n", rcpt.fileIndex)
			fmt.Printf("%s\n", EncodeToString(fileKey))

			var line string
			// Should handle Scan returning false?
			for scanner.Scan() {
				line = scanner.Text()
				if len(line) != 0 {
					break
				}
			}
			if line != "-> ok" {
				return fmt.Errorf("unexpected response to file-key: %s", line)
			}

			// we successfully unwrapped using this id, so stop
			break
		}
	}

	fmt.Printf("-> done\n\n")

	return nil
}

var b64 = base64.RawStdEncoding.Strict()

func DecodeString(s string) ([]byte, error) {
	// CR and LF are ignored by DecodeString, but we don't want any malleability.
	if strings.ContainsAny(s, "\n\r") {
		return nil, errors.New(`unexpected newline character`)
	}
	return b64.DecodeString(s)
}

var EncodeToString = b64.EncodeToString
