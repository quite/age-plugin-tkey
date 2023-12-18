package identity

import (
	"bytes"
	"crypto/ecdh"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/quite/age-plugin-tkey/internal/tkey"
	"github.com/quite/tkeyx25519"
	"github.com/tillitis/tkeyclient"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	ErrWrongDevice = constError("wrong TKey or different tkey-device-x25519 app")
)

const (
	identitySize       = tkeyx25519.UserSecretSize + 1 + pubKeyHashPartSize
	pubKeyHashPartSize = 2
	// These areas defined in age:
	x25519Label = "age-encryption.org/v1/X25519"
	fileKeySize = 16
)

type Identity struct {
	userSecret   []byte
	requireTouch bool
	pubKey       []byte
}

func NewIdentity(userSecret []byte, requireTouch bool) (*tkeyclient.UDI, *Identity, error) {
	udi, pubKey, err := tkey.GetPubKey(userSecret, requireTouch)
	if err != nil {
		return nil, nil, fmt.Errorf("GetPubKey failed: %w", err)
	}
	if l := len(pubKey); l != curve25519.PointSize {
		return nil, nil, fmt.Errorf("pubKey is %d bytes, expected %d", l, curve25519.PointSize)
	}
	return udi, &Identity{
		userSecret:   userSecret,
		requireTouch: requireTouch,
		pubKey:       pubKey,
	}, nil
}

func NewIdentityFromRawID(rawID []byte) (*Identity, error) {
	userSecret, requireTouch, pubKeyHashPart, err := parseRawID(rawID)
	if err != nil {
		return nil, fmt.Errorf("parseBytes failed: %w", err)
	}

	_, pubKey, err := tkey.GetPubKey(userSecret, requireTouch)
	if err != nil {
		return nil, fmt.Errorf("GetPubKey failed: %w", err)
	}

	newHash := blake2s.Sum256(pubKey)
	if !bytes.Equal(newHash[:pubKeyHashPartSize], pubKeyHashPart) {
		return nil, ErrWrongDevice
	}

	return &Identity{
		userSecret:   userSecret,
		requireTouch: requireTouch,
		pubKey:       pubKey,
	}, nil
}

func (id *Identity) EncodeIdentity(pluginName string) (string, error) {
	identityStr := plugin.EncodeIdentity(pluginName, id.bytes())
	if identityStr == "" {
		return "", fmt.Errorf("EncodeIdentity returned empty string")
	}
	return identityStr, nil
}

func (id *Identity) EncodeRecipient() (string, error) {
	pubKeyStruct, err := ecdh.X25519().NewPublicKey(id.pubKey)
	if err != nil {
		return "", fmt.Errorf("NewPublicKey failed: %w", err)
	}

	recipientStr, err := plugin.EncodeX25519Recipient(pubKeyStruct)
	if err != nil {
		return "", fmt.Errorf("EncodeX25519Recipient failed: %w", err)
	}

	return recipientStr, nil
}

func (id *Identity) Unwrap(pubKey []byte, wrappedFileKey []byte) ([]byte, error) {
	sharedSecret, err := tkey.DoECDH(id.userSecret, id.requireTouch, pubKey)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 0, len(pubKey)+len(id.pubKey))
	salt = append(salt, pubKey...)
	salt = append(salt, id.pubKey...)

	h := hkdf.New(sha256.New, sharedSecret, salt, []byte(x25519Label))
	wrappingKey := make([]byte, chacha20poly1305.KeySize)
	if _, err = io.ReadFull(h, wrappingKey); err != nil {
		return nil, fmt.Errorf("ReadFull failed: %w", err)
	}

	fileKey, err := aeadDecrypt(wrappingKey, fileKeySize, wrappedFileKey)
	if errors.Is(err, errIncorrectCiphertextSize) {
		return nil, errors.New("invalid X25519 recipient block: incorrect file key size")
	} else if err != nil {
		return nil, age.ErrIncorrectIdentity
	}

	return fileKey, nil
}

func (id *Identity) RequireTouch() bool {
	return id.requireTouch
}

func (id *Identity) bytes() []byte {
	var buf bytes.Buffer

	buf.Write(id.userSecret)

	if id.requireTouch {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}

	hash := blake2s.Sum256(id.pubKey)
	buf.Write(hash[:pubKeyHashPartSize])

	if l := buf.Len(); l != identitySize {
		// This won't happen
		panic(fmt.Sprintf("%d bytes, expected %d", l, identitySize))
	}

	return buf.Bytes()
}

func parseRawID(bytes []byte) ([]byte, bool, []byte, error) {
	if l := len(bytes); l != identitySize {
		return nil, false, nil, fmt.Errorf("%d bytes, expected %d", l, identitySize)
	}

	var offset uint
	userSecret := bytes[offset : offset+tkeyx25519.UserSecretSize]
	offset += tkeyx25519.UserSecretSize
	requireTouch := bytes[offset] == 1
	offset += 1
	pubKeyHashPart := bytes[offset : offset+pubKeyHashPartSize]

	return userSecret, requireTouch, pubKeyHashPart, nil
}

var errIncorrectCiphertextSize = errors.New("encrypted value has unexpected length")

func aeadDecrypt(key []byte, size int, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) != (size + aead.Overhead()) {
		return nil, errIncorrectCiphertextSize
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	b, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aead.Open failed: %w", err)
	}
	return b, nil
}

type constError string

func (err constError) Error() string {
	return string(err)
}
