package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/quite/age-plugin-tkey/internal/tkey"
)

const (
	pluginName = "tkey"
)

var version = "0.0.2"

// if AGEDEBUG=plugin then age sends plugin's stderr (and own debug)
// to stderr
var le = log.New(os.Stderr, "", 0)

var (
	generateFlag, noTouchFlag, convertFlag, versionFlag bool
	agePluginFlag, outputFlag                           string
)

func main() {
	// TODO --uss ?
	flag.StringVar(&agePluginFlag, "age-plugin", "", "For choosing state machine.")
	descGenerate := "Generate an identity backed by TKey."
	descOutput := "Output generated identity to file at PATH."
	descNoTouch := "Generate an identity for which the TKey will NOT require physical touch before computing a shared key (doing X25519 ECDH)."
	descConvert := "Convert TKey identities to recipients, reading from stdin and writing to stdout. The same TKey used when generating the identities must be plugged in. Useful if you loose the recipient (comment)."
	descVersion := "Output version information and exit."
	flag.BoolVar(&generateFlag, "generate", false, descGenerate)
	flag.BoolVar(&generateFlag, "g", false, descGenerate)
	flag.StringVar(&outputFlag, "output", "", descOutput)
	flag.StringVar(&outputFlag, "o", "", descOutput)
	flag.BoolVar(&noTouchFlag, "no-touch", false, descNoTouch)
	flag.BoolVar(&convertFlag, "y", false, descConvert)
	flag.BoolVar(&versionFlag, "version", false, descVersion)
	flag.Usage = func() {
		le.Printf(`Usage:
  age-plugin-tkey [OPTIONS]

Options:
  -g, --generate     %s
  -o, --output PATH  %s
  --no-touch         %s
  -y                 %s
  --version          %s

Examples:
  $ age-plugin-tkey -g -o tkeyids
  recipient: age1ts5c032h8l6eykkum0jt2clxgtztv8gwu7aamj0mwcx4ewwelcks3s93ru

  $ age-plugin-tkey -y <tkeyids
  age1ts5c032h8l6eykkum0jt2clxgtztv8gwu7aamj0mwcx4ewwelcks3s93ru
`, descGenerate, descOutput, wrap(descNoTouch, 80-21, 21), wrap(descConvert, 80-21, 21), descVersion)
	}
	flag.Parse()

	if len(flag.Args()) > 0 {
		le.Printf("Unexpected positional argument(s)\n")
		flag.Usage()
		os.Exit(2)
	}

	if versionFlag {
		fmt.Printf(`age-plugin-tkey %s

Embedded tkey-device-x25519 app binary:
filename: %s
sha512sum: %s
`, version, tkey.AppFile, tkey.AppHash)
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

	if !generateFlag && !convertFlag && agePluginFlag == "" {
		flag.Usage()
		return 0
	}

	passed := 0
	if generateFlag {
		passed++
	}
	if convertFlag {
		passed++
	}
	if agePluginFlag != "" {
		passed++
	}
	if passed > 1 {
		le.Printf("Only one of -g, -y, and --age-plugin can be used\n")
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
		if !generate(out, noTouchFlag == false) {
			return 1
		}
		return 0
	}

	if convertFlag {
		if !convert(os.Stdin, os.Stdout) {
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
