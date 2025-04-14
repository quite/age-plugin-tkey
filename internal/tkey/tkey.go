package tkey

import (
	"bytes"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/quite/tkeyx25519"
	"github.com/tillitis/tkeyclient"
	"golang.org/x/crypto/curve25519"
)

const (
	ErrWrongDeviceApp = constError("wrong device app")
)

var (
	AppHash string
	AppFile = "unknown"
)

const (
	pluginDomain = "tillitis.se/tkey"
	verbose      = false
)

var (
	le = log.New(os.Stderr, "", 0)
	//go:embed x25519-v0.0.2.bin
	appBin []byte
	//go:embed x25519-hashes.sha512
	hashes []byte
)

func init() {
	ah := sha512.Sum512(appBin)
	AppHash = hex.EncodeToString(ah[:])
	lines := strings.Split(string(hashes), "\n")
	for _, l := range lines {
		ss := strings.Split(l, " ")
		if len(ss) != 3 {
			continue
		}
		h, err := hex.DecodeString(ss[0])
		if err != nil {
			log.Fatal(err)
		}
		if bytes.Equal(h, ah[:]) {
			AppFile = ss[2]
		}
	}
}

func GetPubKey(userSecret []byte, requireTouch bool) (*tkeyclient.UDI, []byte, error) {
	if l := len(userSecret); l != tkeyx25519.UserSecretSize {
		return nil, nil, fmt.Errorf("userSecret is %d bytes, expected %d", l, tkeyx25519.UserSecretSize)
	}

	t := tkey{}
	udi, err := t.connect(verbose)
	if err != nil {
		return nil, nil, fmt.Errorf("connect failed: %w", err)
	}
	defer t.disconnect()

	pubKey, err := t.x25519.GetPubKey(pluginDomain, [tkeyx25519.UserSecretSize]byte(userSecret),
		requireTouch)
	if err != nil {
		return nil, nil, fmt.Errorf("GetPubKey failed: %w", err)
	}

	return udi, pubKey, nil
}

func DoECDH(userSecret []byte, requireTouch bool, theirPubKey []byte) ([]byte, error) {
	if l := len(userSecret); l != tkeyx25519.UserSecretSize {
		return nil, fmt.Errorf("userSecret is %d bytes, expected %d", l, tkeyx25519.UserSecretSize)
	}
	if l := len(theirPubKey); l != curve25519.PointSize {
		return nil, fmt.Errorf("theirPubKey is %d bytes, expected %d", l, curve25519.PointSize)
	}

	t := tkey{}
	if _, err := t.connect(verbose); err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	defer t.disconnect()

	shared, err := t.x25519.DoECDH(pluginDomain, [tkeyx25519.UserSecretSize]byte(userSecret),
		requireTouch, [curve25519.PointSize]byte(theirPubKey))
	if err != nil {
		return nil, fmt.Errorf("DoECDH failed: %w", err)
	}

	return shared, nil
}

type tkey struct {
	x25519 tkeyx25519.X25519
}

const (
	wantFWName0  = "tk1 "
	wantFWName1  = "mkdf"
	wantAppName0 = "x255"
	wantAppName1 = "19  "
)

func (t *tkey) connect(verbose bool) (*tkeyclient.UDI, error) {
	tkeyclient.SilenceLogging()

	devPath := os.Getenv("AGE_TKEY_PORT")
	if devPath == "" {
		var err error
		devPath, err = tkeyclient.DetectSerialPort(verbose)
		if err != nil {
			return nil, fmt.Errorf("DetectSerialPort failed: %w", err)
		}
	}

	tk := tkeyclient.New()
	if verbose {
		le.Printf("Connecting to device on serial port %s...\n", devPath)
	}
	if err := tk.Connect(devPath); err != nil {
		// nolint:staticcheck // ignoring ST1005, to capitalize func name
		return nil, fmt.Errorf("Connect %s failed: %w", devPath, err)
	}

	t.x25519 = tkeyx25519.New(tk)

	// TODO handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	var udi *tkeyclient.UDI
	if isFirmwareMode(tk) {
		if verbose {
			le.Printf("Device is in firmware mode. Loading app...\n")
		}
		var err error
		udi, err = tk.GetUDI()
		if err != nil {
			return nil, fmt.Errorf("GetUDI failed: %w", err)
		}
		if err := tk.LoadApp(appBin, []byte{}); err != nil {
			t.disconnect()
			return nil, fmt.Errorf("LoadApp failed: %w", err)
		}
	}

	if !isWantedApp(t.x25519) {
		if verbose {
			le.Printf("The TKey may already be running an app, but not the expected x25519-app.\n" +
				"Please unplug and plug it in again.\n")
		}
		t.disconnect()
		return nil, ErrWrongDeviceApp
	}

	return udi, nil
}

func (t *tkey) disconnect() {
	if err := t.x25519.Close(); err != nil {
		le.Printf("%s\n", err)
	}
}

// func handleSignals(action func(), sig ...os.Signal) {
// 	ch := make(chan os.Signal, 1)
// 	signal.Notify(ch, sig...)
// 	go func() {
// 		for {
// 			<-ch
// 			action()
// 		}
// 	}()
// }

func isFirmwareMode(tk *tkeyclient.TillitisKey) bool {
	nameVer, err := tk.GetNameVersion()
	if err != nil {
		if !errors.Is(err, io.EOF) && !errors.Is(err, tkeyclient.ErrResponseStatusNotOK) {
			le.Printf("GetNameVersion failed: %s\n", err)
		}
		return false
	}
	// not caring about nameVer.Version
	return nameVer.Name0 == wantFWName0 &&
		nameVer.Name1 == wantFWName1
}

func isWantedApp(x25519 tkeyx25519.X25519) bool {
	nameVer, err := x25519.GetAppNameVersion()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			le.Printf("GetAppNameVersion: %s\n", err)
		}
		return false
	}
	// not caring about nameVer.Version
	return nameVer.Name0 == wantAppName0 &&
		nameVer.Name1 == wantAppName1
}

type constError string

func (err constError) Error() string {
	return string(err)
}
