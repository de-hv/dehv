package ui

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

func newErrWriter() io.Writer {
	return colorprofile.NewWriter(Stderr, os.Environ())
}
