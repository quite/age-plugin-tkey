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
	"github.com/quite/age-plugin-tkey/internal/tkey"
	"github.com/tillitis/tkeyclient"
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
	rawIdentities := [][]byte{}
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

			rawIdentities = append(rawIdentities, idBytes)

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

	if len(rawIdentities) == 0 {
		return fmt.Errorf("no identities specified")
	}

	identities, err := tryIdentities(rawIdentities, r)
	if err != nil {
		return err
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

func tryIdentities(rawIdentities [][]byte, r *bufio.Reader) ([]*identity.Identity, error) {
	var identities []*identity.Identity

	for _, idBytes := range rawIdentities {
		id, err := tryIdentity(idBytes, r)
		if err != nil {
			return nil, err
		}

		if id != nil {
			identities = append(identities, id)
		}
	}

	return identities, nil
}

// tryIdentity returns (nil, nil) when the identity could not be
// "opened" but this was not deemed a fatal error
func tryIdentity(idBytes []byte, r *bufio.Reader) (*identity.Identity, error) {
tryAgain:
	id, err := identity.NewIdentityFromBytes(idBytes)
	if err != nil {
		// TODO? we only do confirm if no device, or wrong device app
		if !errors.Is(err, tkeyclient.ErrNoDevice) && !errors.Is(err, tkey.ErrWrongDeviceApp) {
			le.Printf("identity skipped: NewIdentityFromBytes failed: %s\n", err)
			return nil, nil
		}

		// TODO works because len(msg) < 48 -- we should wrap the base64!
		msg := "Please plug in your TKey"
		if errors.Is(err, tkey.ErrWrongDeviceApp) {
			msg = "TKey is running wrong app, please reconnect it"
		}

		fmt.Printf("-> confirm %s %s\n", EncodeToString([]byte("Try again")),
			EncodeToString([]byte("Cancel")))
		fmt.Printf("%s\n", EncodeToString([]byte(msg)))

		s, err := readStanza(r)
		if err != nil {
			return nil, fmt.Errorf("readStanza confirm response failed: %w", err)
		}

		switch s.typ {
		case "ok":
			if len(s.args) != 1 || len(s.data) > 0 {
				return nil, fmt.Errorf("malformed confirm response stanza: %q", s)
			}
			switch s.args[0] {
			case "yes":
				goto tryAgain
			case "no":
				return nil, nil
			default:
				return nil, fmt.Errorf("malformed confirm response stanza: %q", s)
			}

		case "fail":
			if len(s.args) != 0 || len(s.data) > 0 {
				return nil, fmt.Errorf("malformed confirm response stanza: %q", s)
			}
			return nil, nil

		default:
			return nil, fmt.Errorf("malformed confirm response stanza: %q", s)
		}
	}

	return id, nil
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
