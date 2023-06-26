package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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

type stanza struct {
	typ  string
	args []string
	data []byte
}

func runIdentity() error {
	identities := []*identity.Identity{}
	recipients := []*recipient{}

	r := bufio.NewReader(os.Stdin)

	for {
		s, err := readStanza(r)
		if err != nil {
			return fmt.Errorf("readStanza failed: %w", err)
		}
		if s == nil {
			// no more stanzas
			break
		}

		switch s.typ {
		case "add-identity":
			if len(s.args) != 1 || len(s.data) > 0 {
				return fmt.Errorf("malformed add-identity stanza: %q", s)
			}

			name, idBytes, err := plugin.ParseIdentity(s.args[0])
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
			if len(s.args) != 3 || len(s.data) == 0 {
				return fmt.Errorf("malformed recipient-stanza: %q", s)
			}

			fileIndex, recipientType, recipientPubKeyStr := s.args[0], s.args[1], s.args[2]
			if recipientType != "X25519" {
				le.Printf("recipient skipped: type is %s, expected X25519\n", recipientType)
				continue
			}

			// gentle reminder: this pubkey is ephemeral, not sender's identity
			recipientPubKey, err := DecodeString(recipientPubKeyStr)
			if err != nil {
				return fmt.Errorf("decode recipientPubKey failed: %w", err)
			}
			if len(recipientPubKey) != curve25519.PointSize {
				return fmt.Errorf("recipientPubKey has wrong length")
			}

			recipients = append(recipients, &recipient{
				fileIndex:      fileIndex,
				pubKey:         recipientPubKey,
				wrappedFileKey: s.data,
			})
		}

		if s.typ == "done" {
			if len(s.args) != 0 || len(s.data) > 0 {
				return fmt.Errorf("malformed done stanza: %q", s)
			}
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
			// spec says len(fileKey) == 16, so we don't care to wrap
			// base64 at 64 columns (or about final line < 64 columns)
			fmt.Printf("%s\n", EncodeToString(fileKey))

			s, err := readStanza(r)
			if err != nil {
				return fmt.Errorf("readStanza file-key response failed: %w", err)
			}
			if s.typ != "ok" || len(s.args) != 0 || len(s.data) > 0 {
				return fmt.Errorf("malformed file-key response stanza: %q", s)
			}

			// we successfully unwrapped using this id, so stop
			break
		}
	}

	fmt.Printf("-> done\n\n")

	return nil
}

func readStanza(r *bufio.Reader) (*stanza, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			// no more stanzas
			return nil, nil
		}
		return nil, fmt.Errorf("read stanza first-line failed: %w", err)
	}

	s := &stanza{}

	parts := strings.Split(strings.TrimSuffix(string(line), "\n"), " ")
	if len(parts) < 2 || (len(parts) > 0 && parts[0] != "->") {
		return nil, fmt.Errorf("malformed stanza first-line: %q", line)
	}

	s.typ = parts[1]
	s.args = parts[2:] // empty slice if len(parts) == 2

	var encodedData string
	for {
		line, err = r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("stanza data read failed: %w", err)
		}
		encodedData += strings.TrimSuffix(string(line), "\n")
		if len(line) < 64 {
			break
		}
	}

	if len(encodedData) > 0 {
		data, err := DecodeString(encodedData)
		if err != nil {
			return nil, fmt.Errorf("decode stanza data: %w", err)
		}
		s.data = data
	}

	return s, nil
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
