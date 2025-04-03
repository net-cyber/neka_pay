package sms

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// AfroSMS represents configuration for Afro SMS service
type AfroSMS struct {
	Token        string
	IdentifierID string
}

// Response represents the response from AfroSMS API
type Response struct {
	StatusCode int
	Body       string
}

// NewAfroSMS creates a new AfroSMS client
func NewAfroSMS(token, identifierID string) *AfroSMS {
	return &AfroSMS{
		Token:        token,
		IdentifierID: identifierID,
	}
}

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() string {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate a random 6-digit number
	otp := rand.Intn(900000) + 100000

	return fmt.Sprintf("%d", otp)
}

// SendVerificationSMS sends an OTP verification SMS to the provided phone number
func (a *AfroSMS) SendVerificationSMS(phoneNumber, otp string) (*Response, error) {
	message := fmt.Sprintf("Your verification code is: %s. Valid for 10 minutes.", otp)
	return a.SendSMS(phoneNumber, message)
}

// SendSMS sends an SMS to the provided phone number with the given message
func (a *AfroSMS) SendSMS(recipient, message string) (*Response, error) {
	// Build the URL with query parameters
	baseURL := "https://api.afromessage.com/api/send"
	params := url.Values{}
	params.Add("from", a.IdentifierID)
	params.Add("to", recipient)
	params.Add("message", message)

	// Construct the full URL
	fullURL := baseURL + "?" + params.Encode()

	// Create a new request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating SMS request")
		return nil, err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+a.Token)

	// Create HTTP client and send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error sending SMS request")
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Error reading SMS response")
		return nil, err
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}

	// Check for unsuccessful status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"response":   string(body),
		}).Error("SMS API returned an error")
		return response, errors.New("SMS API error: " + string(body))
	}

	logrus.WithFields(logrus.Fields{
		"recipient": recipient,
		"status":    resp.StatusCode,
	}).Info("SMS sent successfully")

	return response, nil
}
