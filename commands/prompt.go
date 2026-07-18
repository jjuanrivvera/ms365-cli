package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// promptLine prints label to stderr and reads one line from stdin (trimmed). It reads one
// byte at a time rather than buffering, so successive prompts on the same reader don't lose
// input a buffered reader would have read ahead and discarded. Use this for non-secret input
// (base URL, y/n confirmations) — never fmt.Scanln, which echoes and stalls on long pastes.
func promptLine(cmd *cobra.Command, label string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), label)
	r := cmd.InOrStdin()
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			b.WriteByte(buf[0])
		}
		if err != nil {
			if b.Len() == 0 {
				return "", err
			}
			break
		}
	}
	return strings.TrimSpace(b.String()), nil
}
