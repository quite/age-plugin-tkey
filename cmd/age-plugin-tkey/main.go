package main

import (
	"bufio"
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/term"
)

const (
	pluginName   = "tkey"
	pluginDomain = "tillitis.se/tkey"
	domainSize   = 78
)

var domain = func() [domainSize]byte {
	var d [domainSize]byte
	if len(d) > domainSize {
		panic("built-in pluginDomain is too long")
	}
	copy(d[:], pluginDomain)
	return d
}()

// if AGEDEBUG=plugin then age sends plugin's stderr (and own debug)
// to stderr
var le = log.New(os.Stderr, "", 0)

func main() {
	var generateOnly, requireTouch bool
	var agePlugin string
	flag.StringVar(&agePlugin, "age-plugin", "", "For choosing state machine")
	descGenerate := "Generate an identity backed by TKey"
	flag.BoolVar(&generateOnly, "generate", false, descGenerate)
	flag.BoolVar(&generateOnly, "g", false, descGenerate)
	flag.BoolVar(&requireTouch, "touch", false, "Require physical touch of TKey upon use of identity")
	flag.Usage = func() {
		le.Printf(`Usage:
  --age-plugin string    For choosing state machine
  -g, --generate         Generate an identity backed by TKey
  --touch                Make the identity require physical touch of TKey
                         upon X25519 key exchange (use with --generate)
`)
	}
	flag.Parse()

	if !generateOnly && agePlugin == "" {
		flag.Usage()
		os.Exit(0)
	}

	if generateOnly {
		if !generate(requireTouch) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if agePlugin != "" {
		switch agePlugin {
		case "identity-v1":
			err := runIdentity()
			if err != nil {
				le.Printf("runIdentity failed: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		default:
			le.Printf("unknown state machine\n")
			os.Exit(1)
		}
	}
}

const (
	identityDataSize = userSecretSize + 1 + pubHashPartSize
	userSecretSize   = 16
	pubHashPartSize  = 2
)

type identityData struct {
	userSecret   [userSecretSize]byte
	requireTouch bool
	pubBytes     [32]byte
}

func generate(requireTouch bool) bool {
	// Generate a privkey on the TKey and get hold of the pubkey

	var userSecret [userSecretSize]byte
	if _, err := rand.Read(userSecret[:]); err != nil {
		le.Printf("rand.Read failed: %s\n", err)
		return false
	}

	pubBytes, err := getPubKey(userSecret, requireTouch)
	if err != nil {
		le.Printf("%s\n", err)
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

	// TODO maybe make our type to manage encode/decode
	var data bytes.Buffer
	data.Write(userSecret[:])
	if requireTouch {
		data.WriteByte(1)
	} else {
		data.WriteByte(0)
	}
	pubHash := blake2s.Sum256(pubBytes)
	data.Write(pubHash[:pubHashPartSize])

	if l := len(data.Bytes()); l != identityDataSize {
		le.Printf("data is %d bytes, expected %d\n", l, identityDataSize)
		return false
	}

	identity := plugin.EncodeIdentity(pluginName, data.Bytes())
	if identity == "" {
		le.Printf("EncodeIdentity returned empty string\n")
		return false
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		le.Printf("recipient: %s\n", recipient)
	}

	fmt.Printf("# created: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("# recipient: %s\n", recipient)
	fmt.Printf("# touch required: %t\n", requireTouch)
	fmt.Printf("%s\n", identity)

	return true
}

const (
	x25519Label = "age-encryption.org/v1/X25519"
	fileKeySize = 16
)

func runIdentity() error {
	identities := []identityData{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		entry := scanner.Text()
		if len(entry) == 0 {
			continue
		}

		entry = strings.TrimPrefix(entry, "-> ")
		cmd := strings.SplitN(entry, " ", 2)

		switch cmd[0] {
		case "add-identity":
			name, data, err := plugin.ParseIdentity(cmd[1])
			if err != nil {
				le.Printf("identity skipped, ParseIdentity failed: %s\n", err)
				continue
			}
			if name != pluginName {
				le.Printf("identity skipped, unknown name\n")
				continue
			}
			if len(data) != identityDataSize {
				le.Printf("identity skipped, unexpected len\n")
				continue
			}

			ourData := identityData{}
			copy(ourData.userSecret[:], data[:userSecretSize])
			ourData.requireTouch = false
			if data[userSecretSize] == 1 {
				ourData.requireTouch = true
			}
			pubHashPart := data[userSecretSize+1 : userSecretSize+1+pubHashPartSize]

			pubBytes, err := getPubKey(ourData.userSecret, ourData.requireTouch)
			if err != nil {
				le.Printf("identity skipped, getPubKey failed: %s\n", err)
				continue
			}
			copy(ourData.pubBytes[:], pubBytes)

			// le.Printf("usersecret: %0x\n", ourData.userSecret)
			// le.Printf("requiretouch: %t\n", ourData.requireTouch)
			// le.Printf("pubbytes: %0x\n", ourData.pubBytes)

			pubHashAgain := blake2s.Sum256(pubBytes)
			if !bytes.Equal(pubHashPart, pubHashAgain[:pubHashPartSize]) {
				le.Printf("identity skipped, hash mismatch\n")
				continue
			}

			identities = append(identities, ourData)
			// le.Printf("Added an identity\n")

		case "recipient-stanza":
			stanza := strings.Split(entry, " ")
			if stanza[2] != "X25519" {
				le.Printf("recipient-stanza skipped, unexpected type\n")
				continue
			}

			pubKey, err := DecodeString(stanza[3])
			if err != nil {
				// TODO maybe this should error out
				le.Printf("recipient-stanza skipped, DecodeString failed: %s\n", err)
				continue
			}
			if l := len(pubKey); l != curve25519.PointSize {
				// TODO maybe this should error out
				le.Printf("got %d bytes, expected %d\n", l, curve25519.PointSize)
				continue
			}

			var wrappedKeyStr string
			for scanner.Scan() {
				entry = scanner.Text()
				wrappedKeyStr += entry
				if len(entry) < 64 {
					break
				}
			}
			wrappedKey, err := DecodeString(wrappedKeyStr)
			if err != nil {
				return fmt.Errorf("DecodeString failed: %w", err)
			}

			if len(identities) == 0 {
				le.Printf("No identities\n")
				continue
			}

			// TODO handle multiple?
			ourData := identities[0]

			sharedSecret, err := computeShared(ourData.userSecret, ourData.requireTouch, [32]byte(pubKey))
			if err != nil {
				// TODO maybe this should error out
				le.Printf("computeShared failed: %s", err)
				continue
			}

			salt := make([]byte, 0, len(pubKey)+len(ourData.pubBytes))
			salt = append(salt, pubKey...)
			salt = append(salt, ourData.pubBytes[:]...)

			h := hkdf.New(sha256.New, sharedSecret, salt, []byte(x25519Label))
			wrappingKey := make([]byte, chacha20poly1305.KeySize)
			if _, err = io.ReadFull(h, wrappingKey); err != nil {
				return fmt.Errorf("ReadFull failed: %w", err)
			}

			fileKey, err := aeadDecrypt(wrappingKey, fileKeySize, wrappedKey)
			if err == errIncorrectCiphertextSize {
				return errors.New("invalid X25519 recipient block: incorrect file key size")
			} else if err != nil {
				return age.ErrIncorrectIdentity
			}

			fmt.Printf("-> file-key 0\n")
			fmt.Printf("%s\n", EncodeToString(fileKey))

		case "done":
			fmt.Printf("-> done\n\n")
		}
	}

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

var errIncorrectCiphertextSize = errors.New("encrypted value has unexpected length")

func aeadDecrypt(key []byte, size int, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) != size+aead.Overhead() {
		return nil, errIncorrectCiphertextSize
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	return aead.Open(nil, nonce, ciphertext, nil)
}
