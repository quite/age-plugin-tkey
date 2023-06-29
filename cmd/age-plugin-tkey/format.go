package main

import (
	"encoding/base64"
	"errors"
	"strings"
)

var b64 = base64.RawStdEncoding.Strict()

func DecodeString(s string) ([]byte, error) {
	// CR and LF are ignored by DecodeString, but we don't want any malleability.
	if strings.ContainsAny(s, "\n\r") {
		return nil, errors.New(`unexpected newline character`)
	}
	return b64.DecodeString(s)
}

var EncodeToString = b64.EncodeToString

const (
	ColumnsPerLine = 64
	BytesPerLine   = ColumnsPerLine / 4 * 3
)

func EncodeToBody(bytes []byte) string {
	if len(bytes) == 0 {
		return "\n"
	}

	var wrapped string
	var lastWasLong bool

	for len(bytes) > 0 {
		var part []byte
		if len(bytes) >= BytesPerLine {
			part, bytes = bytes[:BytesPerLine], bytes[BytesPerLine:]
			lastWasLong = true
		} else {
			part = bytes
			bytes = nil
			lastWasLong = false
		}
		wrapped += b64.EncodeToString(part)
		wrapped += "\n"
	}

	if lastWasLong {
		wrapped += "\n"
	}

	return wrapped
}
