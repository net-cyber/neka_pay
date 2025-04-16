package bankapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SampleBankConfig holds the configuration for the sample bank API
type SampleBankConfig struct {
	APIURL     string
	APIKey     string
	APISecret  string
	MerchantID string
	Timeout    time.Duration
}

// SampleBankAPI implements the BankAPI interface for a sample bank
type SampleBankAPI struct {
	client  *http.Client
	config  SampleBankConfig
	baseURL string
}

// NewSampleBankAPI creates a new API client for the sample bank
func NewSampleBankAPI(config SampleBankConfig) *SampleBankAPI {
	// Set default timeout if not provided
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SampleBankAPI{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config:  config,
		baseURL: config.APIURL,
	}
}

// sampleBankTransferRequest is the specific request format for this bank
type sampleBankTransferRequest struct {
	MerchantID     string `json:"merchant_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	SourceAccount  string `json:"source_account"`
	DestAccount    string `json:"destination_account"`
	DestBankCode   string `json:"destination_bank_code"`
	RecipientName  string `json:"recipient_name"`
	TransactionRef string `json:"transaction_reference"`
	Description    string `json:"description"`
}

// sampleBankTransferResponse is the specific response format from this bank
type sampleBankTransferResponse struct {
	Success       bool   `json:"success"`
	ResponseCode  string `json:"response_code"`
	Message       string `json:"message"`
	TransactionID string `json:"transaction_id"`
	Fee           int64  `json:"fee"`
	Status        string `json:"status"`
}

// TransferMoney sends money to an external bank account
func (b *SampleBankAPI) TransferMoney(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	// Map generic request to the specific bank's request format
	bankReq := sampleBankTransferRequest{
		MerchantID:     b.config.MerchantID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		SourceAccount:  req.FromAccountNumber,
		DestAccount:    req.ToAccountNumber,
		DestBankCode:   req.ToBankCode,
		RecipientName:  req.RecipientName,
		TransactionRef: req.Reference,
		Description:    req.Description,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(bankReq)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal bank transfer request")
		return nil, ErrInvalidRequest
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/transfer", b.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to create HTTP request")
		return nil, ErrBankAPINotAvailable
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.config.APIKey))
	httpReq.Header.Set("X-API-Secret", b.config.APISecret)

	// Send request
	resp, err := b.client.Do(httpReq)
	if err != nil {
		logrus.WithError(err).Error("Failed to send bank transfer request")
		return nil, ErrBankAPINotAvailable
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read bank transfer response")
		return nil, ErrBankAPINotAvailable
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Bank API returned error")
		return nil, ErrTransferFailed
	}

	// Parse response
	var bankResp sampleBankTransferResponse
	if err := json.Unmarshal(body, &bankResp); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal bank transfer response")
		return nil, ErrBankAPINotAvailable
	}

	// Check for API-level errors
	if !bankResp.Success {
		logrus.WithFields(logrus.Fields{
			"response_code": bankResp.ResponseCode,
			"message":       bankResp.Message,
		}).Error("Bank transfer failed")
		return nil, ErrTransferFailed
	}

	// Map bank-specific response to generic response
	return &TransferResponse{
		TransactionID:   bankResp.TransactionID,
		Status:          bankResp.Status,
		Message:         bankResp.Message,
		TransactionFees: bankResp.Fee,
	}, nil
}

// ValidateAccount checks if an account exists at the bank
func (b *SampleBankAPI) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	// Create request URL
	url := fmt.Sprintf("%s/account/validate?bank_code=%s&account_number=%s", b.baseURL, bankCode, accountNumber)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create account validation request")
		return "", ErrBankAPINotAvailable
	}

	// Add headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.config.APIKey))
	req.Header.Set("X-API-Secret", b.config.APISecret)

	// Send request
	resp, err := b.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to send account validation request")
		return "", ErrBankAPINotAvailable
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read account validation response")
		return "", ErrBankAPINotAvailable
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Bank API returned error")
		return "", ErrAccountNotFound
	}

	// Parse response
	var response struct {
		Success      bool   `json:"success"`
		AccountName  string `json:"account_name"`
		ResponseCode string `json:"response_code"`
		Message      string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal account validation response")
		return "", ErrBankAPINotAvailable
	}

	// Check for API-level errors
	if !response.Success {
		logrus.WithFields(logrus.Fields{
			"response_code": response.ResponseCode,
			"message":       response.Message,
		}).Error("Account validation failed")
		return "", ErrAccountNotFound
	}

	return response.AccountName, nil
}

// GetTransactionStatus checks the status of a previous transaction
func (b *SampleBankAPI) GetTransactionStatus(ctx context.Context, transactionID string) (string, error) {
	// Create request URL
	url := fmt.Sprintf("%s/transaction/%s/status", b.baseURL, transactionID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create transaction status request")
		return "", ErrBankAPINotAvailable
	}

	// Add headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.config.APIKey))
	req.Header.Set("X-API-Secret", b.config.APISecret)

	// Send request
	resp, err := b.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to send transaction status request")
		return "", ErrBankAPINotAvailable
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read transaction status response")
		return "", ErrBankAPINotAvailable
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Bank API returned error")
		return "", ErrTransferFailed
	}

	// Parse response
	var response struct {
		Success      bool   `json:"success"`
		Status       string `json:"status"`
		ResponseCode string `json:"response_code"`
		Message      string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal transaction status response")
		return "", ErrBankAPINotAvailable
	}

	// Check for API-level errors
	if !response.Success {
		logrus.WithFields(logrus.Fields{
			"response_code": response.ResponseCode,
			"message":       response.Message,
		}).Error("Transaction status check failed")
		return "", ErrTransferFailed
	}

	return response.Status, nil
}
