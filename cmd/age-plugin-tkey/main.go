package main

import (
	"crypto/sha512"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/quite/age-plugin-tkey/internal/tkey"
)

const (
	progName   = "age-plugin-tkey"
	pluginName = "tkey"
)

var version string

// if AGEDEBUG=plugin then age sends plugin's stderr (and own debug)
// to stderr
var le = log.New(os.Stderr, "", 0)

var (
	generateFlag, noTouchFlag, versionFlag bool
	agePluginFlag, outputFlag              string
)

func main() {
	if version == "" {
		version = getBuildInfo()
	}
	deviceAppInfo := fmt.Sprintf("SHA-512 hash of the tkey-device-x25519 app binary that is loaded onto TKey:\n%0x\n", sha512.Sum512(tkey.AppBinary))

	// TODO --uss ?
	flag.StringVar(&agePluginFlag, "age-plugin", "", "For choosing state machine")
	descGenerate := "Generate an identity backed by TKey"
	descOutput := "Output identity to file at PATH"
	descNoTouch := "Make the identity NOT require physical touch of TKey upon X25519 key exchange (use with --generate)"
	descVersion := "Output version information and exit"
	flag.BoolVar(&generateFlag, "generate", false, descGenerate)
	flag.BoolVar(&generateFlag, "g", false, descGenerate)
	flag.StringVar(&outputFlag, "output", "", descOutput)
	flag.StringVar(&outputFlag, "o", "", descOutput)
	flag.BoolVar(&noTouchFlag, "no-touch", false, descNoTouch)
	flag.BoolVar(&versionFlag, "version", false, descVersion)
	flag.Usage = func() {
		le.Printf(`Usage:
  -g, --generate     %s
  -o, --output PATH  %s
  --no-touch         %s
  --version          %s

%s`, descGenerate, descOutput, wrap(descNoTouch, 80-21, 21), descVersion, deviceAppInfo)
	}
	flag.Parse()

	if versionFlag {
		fmt.Printf("%s %s\n\n%s", progName, version, deviceAppInfo)
		os.Exit(0)
	}

	os.Exit(run())
}

func run() int {
	if !generateFlag && (noTouchFlag || outputFlag != "") {
		le.Printf("-o and --no-touch can only be used together with -g\n")
		flag.Usage()
		return 2
	}

	if !generateFlag && agePluginFlag == "" {
		flag.Usage()
		return 0
	}

	if generateFlag && agePluginFlag != "" {
		le.Printf("Cannot only use one of -g and --age-plugin\n")
		flag.Usage()
		return 2
	}

	if generateFlag {
		out := os.Stdout
		if outputFlag != "" {
			f, err := os.OpenFile(outputFlag, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				le.Printf("OpenFile failed: %s\n", err)
				return 1
			}
			defer func() {
				if err := f.Close(); err != nil {
					le.Printf("Close failed: %s\n", err)
				}
			}()
			out = f
		}
		if !generate(out, noTouchFlag) {
			return 1
		}
		return 0
	}

	switch agePluginFlag {
	case "identity-v1":
		if err := runIdentity(); err != nil {
			le.Printf("runIdentity failed: %s\n", err)
			return 1
		}
	default:
		le.Printf("%s: unknown state machine\n", agePluginFlag)
		return 1
	}

	return 0
}

func getBuildInfo() string {
	version := "devel without BuildInfo"
	if info, ok := debug.ReadBuildInfo(); ok {
		sb := strings.Builder{}
		sb.WriteString("devel")
		for _, setting := range info.Settings {
			if strings.HasPrefix(setting.Key, "vcs") {
				sb.WriteString(fmt.Sprintf(" %s=%s", setting.Key, setting.Value))
			}
		}
		version = sb.String()
	}
	return version
}

func wrap(s string, cols int, indent int) string {
	words := strings.Fields(strings.TrimSpace(s))
	if len(words) == 0 {
		return s
	}
	out := words[0]
	left := cols - len(out)
	for _, w := range words[1:] {
		if (1 + len(w)) > left {
			out += "\n" + strings.Repeat(" ", indent) + w
			left = cols - len(w)
			continue
		}
		out += " " + w
		left -= (1 + len(w))
	}
	return out
}
