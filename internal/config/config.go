package config

import "time"

// AppBaseURL is your application's base URL used for callbacks, redirects, etc.
var AppBaseURL = "http://localhost:8080" // change this to your production URL later

// AppJWTConfig stores JWT secret key and expiry settings
var AppJWTConfig = struct {
	SecretKey []byte
	Expiry    time.Duration
}{
	SecretKey: []byte("your_super_secret_jwt_key"), // change this to a strong secret
	Expiry:    24 * time.Hour,                      // token validity duration
}

// --------------------
// BudPay Configuration
// --------------------
// BudPay config
var BudPaySecretKey = "sk_test_yyqnkqcuyacwbbigttjyw0hzzheq5zqjq9e5cbz" // Replace with your secret key
var BudPayBaseURL = "https://api.budpay.com/api/v2"

// --------------------
// Paystack Configuration
// --------------------

// PaystackSecretKey is your Paystack secret key
var PaystackSecretKey = "sk_test_yyqnkqcuyacwbbigttjyw0hzzheq5zqjq9e5cbz" // replace with your Paystack secret key

// PaystackBaseURL is the API base URL for Paystack
var PaystackBaseURL = "https://api.paystack.co"

// SMTP Email config
var (
	SMTPHost = "mail.ciphernet.net"
	SMTPPort = 465
	SMTPUser = "augustinejohn@ciphernet.net"
	SMTPPass = "Bastardfugee@1"
	SMTPFrom = "augustinejohn@ciphernet.net"
	SMTPName = "Ticketing App"
)
