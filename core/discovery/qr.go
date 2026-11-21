package discovery

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/mdp/qrterminal/v3"
	"rsc.io/qr"
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

// QRDataURL encodes text as a PNG data URL that a web view can display.
func QRDataURL(text string) (string, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", fmt.Errorf("discovery: encode qr: %w", err)
	}
	code.Scale = 8
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(code.PNG()), nil
}
