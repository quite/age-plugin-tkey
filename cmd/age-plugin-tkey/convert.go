package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"filippo.io/age/plugin"
	"github.com/quite/age-plugin-tkey/internal/identity"
)

func convert(in io.Reader, out io.Writer) error {
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
			return fmt.Errorf("ParseIdentity failed on line %d: %w", n, err)
		}
		if name != pluginName {
			continue
		}

		id, err := identity.NewIdentityFromRawID(rawID)
		if err != nil {
			return fmt.Errorf("NewIdentityFromRawID failed: %w", err)
		}

		recipient, err := id.EncodeRecipient()
		if err != nil {
			return fmt.Errorf("EncodeRecipient failed: %w", err)
		}
		fmt.Fprintf(out, "%s\n", recipient)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scan failed: %w", err)
	}

	return nil
}
