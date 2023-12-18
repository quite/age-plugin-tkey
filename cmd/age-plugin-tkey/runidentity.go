package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/quite/age-plugin-tkey/internal/identity"
	"github.com/quite/age-plugin-tkey/internal/tkey"
	"github.com/quite/tkeyx25519"
	"github.com/tillitis/tkeyclient"
	"golang.org/x/crypto/curve25519"
)

type recipient struct {
	fileIndex      int
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

			name, rawID, err := plugin.ParseIdentity(s.args[0])
			if err != nil {
				return fmt.Errorf("ParseIdentity failed: %w", err)
			}
			if name != pluginName {
				le.Printf("identity skipped: plugin name is %s, expected %s\n", name, pluginName)
				continue
			}

			rawIdentities = append(rawIdentities, rawID)

		case "recipient-stanza":
			if len(s.args) != 3 || len(s.data) == 0 {
				return fmt.Errorf("malformed recipient-stanza: %q", s)
			}

			fileIndex, err := strconv.Atoi(s.args[0])
			if err != nil {
				return fmt.Errorf("bad recipient-stanza file_index: %w", err)
			}
			typ, pubKeyStr := s.args[1], s.args[2]
			if typ != "X25519" {
				le.Printf("recipient skipped: type is %s, expected X25519\n", typ)
				continue
			}

			// gentle reminder: this pubkey is the ephemeral session
			// key, not recipient pubkey for sender's identity
			pubKey, err := DecodeString(pubKeyStr)
			if err != nil {
				return fmt.Errorf("decode pubkey failed: %w", err)
			}
			if len(pubKey) != curve25519.PointSize {
				return fmt.Errorf("pubkey has wrong length")
			}

			recipients = append(recipients, &recipient{
				fileIndex:      fileIndex,
				pubKey:         pubKey,
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

	unwrapped := make(map[int]struct{})
	for _, rcpt := range recipients {
		if _, ok := unwrapped[rcpt.fileIndex]; ok {
			continue
		}
		for _, id := range identities {

			if id.RequireTouch() {
				writeStanza("msg", nil, []byte("Touch your TKey to confirm decryption"))
				// TODO? we don't care what the response is
				if _, err := readStanza(r); err != nil {
					return fmt.Errorf("readStanza msg response failed: %w", err)
				}
			}

			fileKey, err := id.Unwrap(rcpt.pubKey, rcpt.wrappedFileKey)
			if err != nil {
				if errors.Is(err, age.ErrIncorrectIdentity) {
					continue
				}

				if e := new(tkeyx25519.ResponseStatusNotOKError); errors.As(err, &e) {
					if e.Code() == tkeyx25519.StatusTouchTimeout {
						writeStanza("msg", nil, []byte("TKey not touched, not decrypting using this identity"))
						// TODO? we don't care what the response is
						if _, err = readStanza(r); err != nil {
							return fmt.Errorf("readStanza msg response failed: %w", err)
						}
						continue
					}
				}

				return err
			}

			writeStanza("file-key", []string{strconv.Itoa(rcpt.fileIndex)}, fileKey)

			s, err := readStanza(r)
			if err != nil {
				return fmt.Errorf("readStanza file-key response failed: %w", err)
			}
			if s.typ != "ok" || len(s.args) != 0 || len(s.data) > 0 {
				return fmt.Errorf("malformed file-key response stanza: %q", s)
			}

			unwrapped[rcpt.fileIndex] = struct{}{}
			// we successfully unwrapped using this id, so stop
			break
		}
	}

	writeStanza("done", nil, nil)

	return nil
}

func tryIdentities(rawIdentities [][]byte, r *bufio.Reader) ([]*identity.Identity, error) {
	var identities []*identity.Identity

	for _, rawID := range rawIdentities {
		id, err := tryIdentity(rawID, r)
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
func tryIdentity(rawID []byte, r *bufio.Reader) (*identity.Identity, error) {
tryAgain:
	id, err := identity.NewIdentityFromRawID(rawID)
	if err != nil {
		// TODO? we only do confirm in some specific cases; and should
		// we do all of them?
		var confirmMsg string
		switch {
		case errors.Is(err, tkeyclient.ErrNoDevice):
			confirmMsg = "Please plug in your TKey"
		case errors.Is(err, tkey.ErrWrongDeviceApp):
			confirmMsg = "TKey is running wrong app, please reconnect it"
		case errors.Is(err, identity.ErrWrongDevice):
			confirmMsg = "Maybe wrong TKey or identity created using different tkey-device-x25519 app"
		default:
			le.Printf("identity skipped: NewIdentityFromRawID failed: %s\n", err)
			return nil, nil
		}

		writeStanza("confirm", []string{
			EncodeToString([]byte("Try again")),
			EncodeToString([]byte("Cancel")),
		}, []byte(confirmMsg))

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

const stanzaPrefix = "->"

func writeStanza(typ string, args []string, data []byte) {
	firstLine := fmt.Sprintf("%s %s", stanzaPrefix, typ)
	for _, arg := range args {
		firstLine += fmt.Sprintf(" %s", arg)
	}
	fmt.Fprintf(os.Stdout, "%s\n", firstLine)
	fmt.Fprintf(os.Stdout, "%s", EncodeToBody(data))
}

func readStanza(r *bufio.Reader) (*stanza, error) {
	firstLine, err := r.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			// no more stanzas
			return nil, nil
		}
		return nil, fmt.Errorf("read stanza first-line failed: %w", err)
	}

	s := &stanza{}

	parts := strings.Split(strings.TrimSuffix(string(firstLine), "\n"), " ")
	if len(parts) < 2 || (len(parts) > 0 && parts[0] != stanzaPrefix) {
		return nil, fmt.Errorf("malformed stanza first-line: %q", firstLine)
	}

	s.typ = parts[1]
	s.args = parts[2:] // empty slice if len(parts) == 2

	var encodedData string
	for {
		line, err := r.ReadBytes('\n')
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
