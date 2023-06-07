package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/quite/tkeyx25519"
	"github.com/tillitis/tkeyclient"
)

const verbose = false

func getPubKey(userSecret [userSecretSize]byte, requireTouch bool) ([]byte, error) {
	t := tkey{}
	if err := t.connect(verbose); err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	defer t.disconnect()

	// TODO setting userSecret to fixed (non-random), it looks like the
	// pubkey doesn't change depending on touchRequired when run on
	// hw, but it does in qemu!?
	pubBytes, err := t.x25519.GetPubKey(domain, userSecret, requireTouch)
	if err != nil {
		return nil, fmt.Errorf("GetPubKey failed: %w", err)
	}

	return pubBytes, nil
}

func computeShared(userSecret [userSecretSize]byte, requireTouch bool, theirPubKey [32]byte) ([]byte, error) {
	t := tkey{}
	if err := t.connect(verbose); err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	defer t.disconnect()

	shared, err := t.x25519.ComputeShared(domain, userSecret, requireTouch, theirPubKey)
	if err != nil {
		return nil, fmt.Errorf("ComputeShared failed: %w", err)
	}

	return shared, nil
}

// nolint:typecheck // Avoid lint error when the embedding file is
// missing. Makefile copies the device app binary here ./app.bin
//
//go:embed app.bin
var appBinary []byte

type tkey struct {
	x25519 tkeyx25519.X25519
}

const (
	wantFWName0  = "tk1 "
	wantFWName1  = "mkdf"
	wantAppName0 = "x255"
	wantAppName1 = "19  "
)

func (t *tkey) connect(verbose bool) error {
	tkeyclient.SilenceLogging()

	devPath := os.Getenv("TKEY_PORT")
	if devPath == "" {
		var err error
		devPath, err = tkeyclient.DetectSerialPort(verbose)
		if err != nil {
			return fmt.Errorf("DetectSerialPort failed: %w", err)
		}
	}

	tk := tkeyclient.New()
	if verbose {
		le.Printf("Connecting to device on serial port %s...\n", devPath)
	}
	if err := tk.Connect(devPath); err != nil {
		return fmt.Errorf("Connect %s failed: %w", devPath, err)
	}

	t.x25519 = tkeyx25519.New(tk)

	// TODO handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	if isFirmwareMode(tk) {
		if verbose {
			le.Printf("Device is in firmware mode. Loading app...\n")
		}
		if err := tk.LoadApp(appBinary, []byte{}); err != nil {
			t.disconnect()
			return fmt.Errorf("LoadApp failed: %w", err)
		}
	}

	if !isWantedApp(t.x25519) {
		if verbose {
			le.Printf("The TKey may already be running an app, but not the expected x25519-app.\n" +
				"Please unplug and plug it in again.\n")
		}
		t.disconnect()
		return fmt.Errorf("Wrong app")
	}

	return nil
}

func (t *tkey) disconnect() {
	if err := t.x25519.Close(); err != nil {
		le.Printf("%v\n", err)
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
