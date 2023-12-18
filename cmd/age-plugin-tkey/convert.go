package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"filippo.io/age/plugin"
	"github.com/quite/age-plugin-tkey/internal/identity"
)

func convert(in io.Reader, out io.Writer) bool {
	pluginPrefix := fmt.Sprintf("AGE-PLUGIN-%s-", strings.ToUpper(pluginName))

	scanner := bufio.NewScanner(in)
	var n int
	for scanner.Scan() {
		n++
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, pluginPrefix) {
			le.Printf("skipped a non-TKey identity\n")
			continue
		}

		name, rawID, err := plugin.ParseIdentity(line)
		if err != nil {
			le.Printf("ParseIdentity failed on line %d: %s\n", n, err)
			return false
		}
		if name != pluginName {
			continue
		}

		id, err := identity.NewIdentityFromRawID(rawID)
		if err != nil {
			le.Printf("NewIdentityFromRawID failed: %s\n", err)
			return false
		}

		recipient, err := id.EncodeRecipient()
		if err != nil {
			le.Printf("EncodeRecipient failed: %s\n", err)
			return false
		}
		fmt.Fprintf(out, "%s\n", recipient)
	}

	if err := scanner.Err(); err != nil {
		le.Printf("Scan failed: %s\n", err)
		return false
	}

	return true
}
