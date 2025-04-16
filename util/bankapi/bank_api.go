package bankapi

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrBankAPINotAvailable = errors.New("bank API is not available")
	ErrInvalidRequest      = errors.New("invalid bank transfer request")
	ErrInsufficientFunds   = errors.New("insufficient funds for transfer")
	ErrTransferFailed      = errors.New("bank transfer failed")
	ErrAccountNotFound     = errors.New("account not found at bank")
)

// TransferRequest contains all the information needed to make an external bank transfer
type TransferRequest struct {
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	FromAccountNumber string `json:"from_account_number"`
	ToAccountNumber   string `json:"to_account_number"`
	ToBankCode        string `json:"to_bank_code"`
	RecipientName     string `json:"recipient_name"`
	Reference         string `json:"reference"`
	Description       string `json:"description,omitempty"`
}

// TransferResponse contains the result of a bank transfer operation
type TransferResponse struct {
	TransactionID   string `json:"transaction_id"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	TransactionFees int64  `json:"transaction_fees,omitempty"`
}

// BankAPI defines the interface that all bank API implementations must satisfy
type BankAPI interface {
	// TransferMoney sends money to an external bank account
	TransferMoney(ctx context.Context, req TransferRequest) (*TransferResponse, error)

	// ValidateAccount checks if the account exists at the bank
	ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error)

	// GetTransactionStatus checks the status of a previous transaction
	GetTransactionStatus(ctx context.Context, transactionID string) (string, error)
}

// BankAPIProvider manages different bank API implementations
type BankAPIProvider struct {
	banks map[string]BankAPI
}

// NewBankAPIProvider creates a new provider to manage bank API integrations
func NewBankAPIProvider() *BankAPIProvider {
	return &BankAPIProvider{
		banks: make(map[string]BankAPI),
	}
}

// RegisterBank registers a bank API implementation for a specific bank code
func (p *BankAPIProvider) RegisterBank(bankCode string, api BankAPI) {
	p.banks[bankCode] = api
}

// GetBankAPI returns the bank API implementation for a specific bank code
func (p *BankAPIProvider) GetBankAPI(bankCode string) (BankAPI, error) {
	api, exists := p.banks[bankCode]
	if !exists {
		return nil, ErrBankAPINotAvailable
	}
	return api, nil
}

// TransferMoney performs a money transfer to the specified bank
func (p *BankAPIProvider) TransferMoney(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	api, err := p.GetBankAPI(req.ToBankCode)
	if err != nil {
		return nil, err
	}

	return api.TransferMoney(ctx, req)
}

// ValidateAccount checks if an account exists at the specified bank
func (p *BankAPIProvider) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	api, err := p.GetBankAPI(bankCode)
	if err != nil {
		return "", err
	}

	return api.ValidateAccount(ctx, bankCode, accountNumber)
}

// GetTransactionStatus checks the status of a transaction at the specified bank
func (p *BankAPIProvider) GetTransactionStatus(ctx context.Context, bankCode, transactionID string) (string, error) {
	api, err := p.GetBankAPI(bankCode)
	if err != nil {
		return "", err
	}

	return api.GetTransactionStatus(ctx, transactionID)
}
