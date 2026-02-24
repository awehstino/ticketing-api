package utils

import (
	// "crypto/tls"
	"fmt"

	"log"
	"os"
	"path/filepath"
	"strings"

	"time"

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
	venue := order.Event.Venue
	ticketCode := item.TicketCode
	ticketName := "N/A"
	if item.Ticket != nil {
		ticketName = item.Ticket.Name
	}
	eventURL := fmt.Sprintf("%s/events/%d", config.AppBaseURL, order.EventID)
	currentYear := time.Now().Year()

	// --- Build modern e-ticket email HTML ---
	html := fmt.Sprintf(`<!doctype html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Your Ticket for %s</title>
    <style>
        body { margin: 0; padding: 0; background-color: #f2f4f6; font-family: Arial, sans-serif; -webkit-font-smoothing: antialiased; }
        .email-container { max-width: 600px; margin: 20px auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 15px rgba(0,0,0,0.05); }
        .header { background-color: #2a2a2a; color: #ffffff; padding: 40px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; }
        .content { padding: 30px; }
        .content p { font-size: 16px; line-height: 1.6; color: #333333; }
        .ticket-details { border: 1px solid #e0e0e0; border-radius: 8px; margin-top: 25px; }
        .ticket-details-header { background-color: #f9f9f9; padding: 15px; border-bottom: 1px solid #e0e0e0; }
        .ticket-details-header h2 { margin: 0; font-size: 20px; color: #333; }
        .ticket-details-body { padding: 20px; display: flex; align-items: center; justify-content: space-between; }
        .details-text { flex-grow: 1; }
        .details-text .detail-item { margin-bottom: 12px; font-size: 15px; }
        .details-text .label { font-weight: bold; color: #555; display: inline-block; width: 100px; }
        .qr-code { text-align: center; margin-left: 20px; }
        .qr-code img { width: 120px; height: 120px; }
        .qr-code p { font-weight: bold; letter-spacing: 1px; margin-top: 5px; font-size: 14px; color: #333; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #888888; border-top: 1px solid #e0e0e0; }
        .button { display: inline-block; background-color: #111827; color: #ffffff; padding: 12px 25px; border-radius: 5px; text-decoration: none; font-weight: bold; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <h1>Your Ticket is Confirmed!</h1>
        </div>
        <div class="content">
            <p>Hello %s,</p>
            <p>Thank you for your purchase. We're excited to see you at <strong>%s</strong>. Please find your ticket details below. You can present this email with the QR code at the entrance.</p>
            
            <div class="ticket-details">
                <div class="ticket-details-header">
                    <h2>Your E-Ticket</h2>
                </div>
                <div class="ticket-details-body">
                    <div class="details-text">
                        <div class="detail-item"><span class="label">Event:</span> %s</div>
                        <div class="detail-item"><span class="label">Date:</span> %s</div>
                        <div class="detail-item"><span class="label">Venue:</span> %s</div>
                        <div class="detail-item"><span class="label">Ticket Type:</span> %s</div>
                        <div class="detail-item"><span class="label">Price:</span> %s</div>
                    </div>
                    <div class="qr-code">
                        <img src="cid:qrcode.png" alt="QR Code">
                        <p>%s</p>
                    </div>
                </div>
            </div>

            <p style="text-align:center; margin-top: 30px;">
                A PDF version of your ticket is also attached to this email for your convenience.
            </p>
            <p style="text-align:center;">
                <a href="%s" class="button">View Event Details</a>
            </p>
        </div>
        <div class="footer">
            <p>&copy; %d Ticketing App. All rights reserved.</p>
            <p>If you have any questions, please contact our support team.</p>
        </div>
    </div>
</body>
</html>`,
		eventName, // for title
		buyerName,
		eventName, // for content
		eventName, // for details
		eventDateStr,
		venue,
		ticketName,
		priceStr,
		ticketCode,
		eventURL,
		currentYear,
	)

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
