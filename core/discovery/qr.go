package discovery

import (
	"io"

	"github.com/mdp/qrterminal/v3"
)

// WriteQR renders text as a QR code that terminals can display.
func WriteQR(w io.Writer, text string) error {
	qrterminal.GenerateWithConfig(text, qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     w,
		HalfBlocks: true,
	})
	return nil
}
