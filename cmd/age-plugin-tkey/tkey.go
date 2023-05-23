package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/quite/age-plugin-tkey/internal/util"
	"github.com/quite/tkeyx25519"
	"github.com/tillitis/tkeyclient"
)

type tkey struct {
	x25519 tkeyx25519.X25519
}

const (
	wantFWName0  = "tk1 "
	wantFWName1  = "mkdf"
	wantAppName0 = "x255"
	wantAppName1 = "19  "
)

func (t *tkey) connect() error {
	// TODO
	verbose := true

	tkeyclient.SilenceLogging()

	// TODO not allowing for any custom devpath (or speed)
	devPath, err := util.DetectSerialPort(true)
	if err != nil {
		return err
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

func handleSignals(action func(), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		for {
			<-ch
			action()
		}
	}()
}

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
