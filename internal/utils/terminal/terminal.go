// Package terminal contains functions for interacting with the terminal.
package terminal

import (
	"os"
)

// IsTTY returns true if the terminal is a TTY.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
