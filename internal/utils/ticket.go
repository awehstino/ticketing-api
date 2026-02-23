package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
)

// GenerateTicketCode creates a unique human-readable ticket code
func GenerateTicketCode(orderID, ticketID uint) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("TIX-%d-%d-%d", orderID, ticketID, timestamp)
}

// GenerateQRCode saves a QR code image to /storage/qrcodes/
func GenerateQRCode(content, filename string) (string, error) {
	path := "storage/qrcodes/" + filename + ".png"

	// Ensure directory exists
	_ = os.MkdirAll("storage/qrcodes", os.ModePerm)

	err := qrcode.WriteFile(content, qrcode.Medium, 256, path)
	if err != nil {
		return "", err
	}
	return path, nil
}
