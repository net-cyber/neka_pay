package bankapi

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// CBEBankConfig holds the configuration for the CBE bank API
type CBEBankConfig struct {
	APIURL    string
	APIKey    string
	APISecret string
	Timeout   time.Duration
}

// CBEBankAPI implements the BankAPI interface for Commercial Bank of Ethiopia
type CBEBankAPI struct {
	config     CBEBankConfig
	mockData   map[string]string // Map of account numbers to account names
	validRegex *regexp.Regexp    // Regex for validating account numbers
}

// NewCBEBankAPI creates a new API client for CBE bank with mock data
func NewCBEBankAPI(config CBEBankConfig) *CBEBankAPI {
	// In a real implementation, these would come from the bank's API
	mockAccounts := map[string]string{
		"1000123456": "Abebe Kebede",
		"1000567890": "Tigist Haile",
		"1000246802": "Dawit Mekonnen",
		"1000135792": "Rahel Girmay",
		"1000987654": "Solomon Tesfaye",
		"1000555555": "Helen Bekele",
		"1000111111": "Daniel Assefa",
		"1000222222": "Meron Tadesse",
		"1000333333": "Yohannes Gebremedhin",
		"1000444444": "Selamawit Tsegaye",
	}

	// Set default timeout if not provided
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &CBEBankAPI{
		config:     config,
		mockData:   mockAccounts,
		validRegex: regexp.MustCompile(`^1000\d{6}$`), // CBE accounts start with 1000 followed by 6 digits
	}
}

// TransferMoney sends money to an external bank account
func (b *CBEBankAPI) TransferMoney(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	// Check if account exists in our mock data
	if _, exists := b.mockData[req.ToAccountNumber]; !exists && !b.validRegex.MatchString(req.ToAccountNumber) {
		return nil, ErrAccountNotFound
	}

	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// Generate a unique transaction ID
	transactionID := fmt.Sprintf("CBE%d%06d", time.Now().Unix(), rand.Intn(1000000))

	// Random fee between 5 and 15
	fee := int64(5 + rand.Intn(11))

	// 5% chance of a random failure
	if rand.Intn(100) < 5 {
		return nil, ErrTransferFailed
	}

	return &TransferResponse{
		TransactionID:   transactionID,
		Status:          "completed",
		Message:         "Transfer completed successfully",
		TransactionFees: fee,
	}, nil
}

// ValidateAccount checks if the account exists at the bank
func (b *CBEBankAPI) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)

	// Check if account exists in our pre-defined mock data
	if name, exists := b.mockData[accountNumber]; exists {
		return name, nil
	}

	

	return "", ErrAccountNotFound
}

// GetTransactionStatus checks the status of a previous transaction
func (b *CBEBankAPI) GetTransactionStatus(ctx context.Context, transactionID string) (string, error) {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

	// Check if the transaction ID starts with CBE prefix
	if len(transactionID) < 3 || transactionID[:3] != "CBE" {
		return "", errors.New("invalid transaction ID format")
	}

	// For demo purposes, most transactions are completed
	statuses := []string{"completed", "completed", "completed", "completed", "completed", "processing", "failed"}
	return statuses[rand.Intn(len(statuses))], nil
}
