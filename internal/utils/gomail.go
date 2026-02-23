package utils

import (
	// "crypto/tls"
	"fmt"

	"log"
	"os"
	"path/filepath"
	"strings"

	// "time"

	"github.com/awehstino/ticketing-api/internal/config"

	"github.com/awehstino/ticketing-api/internal/models"

	gomail "gopkg.in/mail.v2"

	// PDF packages
	"github.com/jung-kurt/gofpdf"
)

// Generic email sender (for reusability)
func SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", config.SMTPName, config.SMTPFrom))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	log.Println("Dialing SMTP server:", config.SMTPHost, config.SMTPPort)
	d := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass)
	// d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // This is insecure and should not be used in production.

	log.Println("Attempting to send email to:", to)
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}
	log.Printf("Email sent successfully to %s", to)
	return nil
}

// GenerateTicketPDF creates a modern raffle-style ticket with a QR code and details.
func GenerateTicketPDF(eventName, buyerName, eventDate, venue, ticketPrice, qrPath, outputPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("cannot create pdf dir: %w", err)
	}

	pdf := gofpdf.New("L", "mm", "A5", "")
	pdf.AddPage()

	// --- Background (Dark teal like sample) ---
	pdf.SetFillColor(35, 60, 63) // #233C3F
	pdf.Rect(0, 0, 210, 148, "F")

	// --- Ticket main panel (white border) ---
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.4)
	pdf.Rect(15, 25, 130, 95, "")

	// --- Ticket Info Section ---
	pdf.SetTextColor(255, 255, 255)

	// Top info
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetXY(25, 38)
	pdf.CellFormat(0, 6, fmt.Sprintf("Ticket No: %s", buyerName), "", 0, "L", false, 0, "")
	pdf.SetXY(110, 38)
	pdf.CellFormat(0, 6, fmt.Sprintf("Price: %s", ticketPrice), "", 0, "R", false, 0, "")

	// Event title (centered)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(25, 65)
	pdf.CellFormat(100, 10, strings.ToUpper(eventName), "", 0, "C", false, 0, "")

	// Draw date and venue
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetXY(25, 100)
	pdf.CellFormat(0, 6, fmt.Sprintf("Draw Date: %s", eventDate), "", 0, "L", false, 0, "")
	pdf.SetXY(80, 100)
	pdf.CellFormat(65, 6, fmt.Sprintf("Venue: %s", venue), "", 0, "R", false, 0, "")

	// --- Dotted perforation line (between main and QR area) ---
	pdf.SetDashPattern([]float64{1, 2}, 0)
	pdf.Line(150, 20, 150, 125)
	pdf.SetDashPattern([]float64{}, 0)

	// --- Right section: QR and label ---
	if qrPath != "" {
		if _, err := os.Stat(qrPath); err == nil {
			pdf.ImageOptions(qrPath, 160, 45, 40, 40, false, gofpdf.ImageOptions{}, 0, "")
		}
	}

	// Text under QR
	pdf.SetXY(160, 90)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(40, 6, "ADMIT ONE", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(40, 5, fmt.Sprintf("Ticket No: %s", buyerName), "", 1, "C", false, 0, "")

	// --- Decorative curved notches (optional) ---
	drawNotches(pdf)

	// --- Save PDF ---
	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return fmt.Errorf("failed writing pdf: %w", err)
	}
	return nil
}

// drawNotches adds small rounded notches to each side (like real raffle perforation)
func drawNotches(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.3)

	// Left notch (top)
	pdf.Arc(15, 30, 3, 3, 0, 180, 360, "D")
	// Left notch (bottom)
	pdf.Arc(15, 120, 3, 3, 0, 180, 360, "D")

	// Right notch (top)
	pdf.Arc(150, 30, 3, 3, 0, 0, 180, "D")
	// Right notch (bottom)
	pdf.Arc(150, 120, 3, 3, 0, 0, 180, "D")
}

// SendTicketEmail builds the HTML email, attaches the PDF and embeds the qrcode image.
// Expects that order.User or order.Guest is present (caller should preload).
// - order: the Order (with User/Guest preloaded)
// - item: the specific OrderItem
// - qrPath: path to QR PNG (must exist)
func SendTicketEmail(order models.Order, item models.OrderItem, qrPath string, pdfPath string) error {
	// Determine recipient and buyer name
	var toEmail, buyerName string
	if order.User != nil && order.User.Email != "" {
		toEmail = order.User.Email
		buyerName = order.User.Name
	} else if order.Guest != nil && order.Guest.Email != "" {
		toEmail = order.Guest.Email
		buyerName = order.Guest.Name
	} else {
		return fmt.Errorf("no recipient found for order %d", order.ID)
	}

	// Prepare ticket info
	eventName := order.Event.Title
	eventDateStr := order.Event.StartTime.Format("02 Jan 2006 15:04")
	priceStr := fmt.Sprintf("₦%.2f", item.UnitPrice)
	ticketCode := item.TicketCode

	// --- Build modern e-ticket email HTML ---
	html := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Your Ticket</title>
<style>
  body{background:#f4f5f7;margin:0;padding:32px;font-family:'Segoe UI',Arial,sans-serif;color:#111827}
  .ticket{max-width:720px;margin:auto;background:#fff;border-radius:14px;overflow:hidden;
    box-shadow:0 8px 30px rgba(0,0,0,0.1);border:1px solid #e5e7eb}
  .header{background:#111827;color:#fff;padding:24px;text-align:center}
  .header h1{margin:0;font-size:24px;letter-spacing:0.3px}
  .content{padding:28px 32px}
  .info{display:flex;justify-content:space-between;flex-wrap:wrap;margin-bottom:24px}
  .info div{margin-bottom:8px}
  .label{font-weight:600;color:#6b7280;font-size:14px}
  .value{font-size:15px;color:#111827}
  .code-box{margin-top:20px;text-align:center;border-top:1px dashed #d1d5db;padding-top:20px}
  .code-box h3{margin:4px 0;color:#111827;font-size:18px;letter-spacing:1px}
  .qr{width:150px;height:150px;border-radius:8px;margin-top:10px}
  .footer{text-align:center;padding:18px;font-size:13px;color:#6b7280;background:#f9fafb;border-top:1px solid #e5e7eb}
  .btn{display:inline-block;margin-top:12px;background:#111827;color:#fff;
    text-decoration:none;padding:10px 16px;border-radius:8px;font-size:14px}
</style>
</head>
<body>
  <div class="ticket">
    <div class="header">
      <h1>%s</h1>
      <p style="margin-top:6px;font-size:14px;color:#9ca3af">E-Ticket Confirmation</p>
    </div>
    <div class="content">
      <p style="font-size:16px;">Hi <strong>%s</strong>,</p>
      <p>Thank you for purchasing your ticket. Below are your ticket details:</p>
      <div class="info">
        <div>
          <div class="label">Event</div>
          <div class="value">%s</div>
        </div>
        <div>
          <div class="label">Date & Time</div>
          <div class="value">%s</div>
        </div>
        <div>
          <div class="label">Ticket Code</div>
          <div class="value"><strong>%s</strong></div>
        </div>
        <div>
          <div class="label">Price</div>
          <div class="value">%s</div>
        </div>
      </div>

      <div class="code-box">
        <img class="qr" src="cid:qrcode.png" alt="QR Code"><br>
        <h3>%s</h3>
        <small style="color:#6b7280;">Scan this code to verify your ticket</small><br>
        <a class="btn" href="#">View Ticket PDF</a>
      </div>
    </div>
    <div class="footer">
      Please bring this ticket (digital or printed) to gain entry. This ticket is personal and non-transferable.
    </div>
  </div>
</body>
</html>`,
		eventName, buyerName, eventName, eventDateStr, ticketCode, priceStr, ticketCode)

	// --- Prepare email ---
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", config.SMTPName, config.SMTPFrom))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", fmt.Sprintf("🎟️ Your Ticket for %s", eventName))
	m.SetBody("text/html", html)

	// Embed QR image
	if qrPath != "" {
		if _, err := os.Stat(qrPath); err == nil {
			m.Embed(qrPath, gomail.SetHeader(map[string][]string{"Content-ID": {"<qrcode.png>"}}))
		}
	}

	// Attach PDF
	if pdfPath != "" {
		if _, err := os.Stat(pdfPath); err == nil {
			m.Attach(pdfPath)
		}
	}

	// Dialer
	d := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass)
	// d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // This is insecure and should not be used in production.

	// Send email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("❌ Failed to send ticket email to %s for order %d: %v", toEmail, order.ID, err)
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
