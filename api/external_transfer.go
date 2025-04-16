package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/token"
	"github.com/net-cyber/neka_pay/util/bankapi"
	"github.com/sirupsen/logrus"
)

type externalTransferRequest struct {
	FromAccountID   int64  `json:"from_account_id" binding:"required,min=1"`
	ToBankCode      string `json:"to_bank_code" binding:"required"`
	ToAccountNumber string `json:"to_account_number" binding:"required"`
	Amount          int64  `json:"amount" binding:"required,gt=0"`
	Currency        string `json:"currency" binding:"required,currency"`
	Description     string `json:"description"`
}

type externalTransferResponse struct {
	ID              int64     `json:"id"`
	FromAccountID   int64     `json:"from_account_id"`
	ToBankCode      string    `json:"to_bank_code"`
	ToAccountNumber string    `json:"to_account_number"`
	RecipientName   string    `json:"recipient_name"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	Reference       string    `json:"reference"`
	Description     string    `json:"description,omitempty"`
	TransactionID   string    `json:"transaction_id,omitempty"`
	TransactionFees int64     `json:"transaction_fees,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// account lookup request and response structs
type accountLookupRequest struct {
	BankCode      string `json:"bank_code" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
}

type accountLookupResponse struct {
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	Found         bool   `json:"found"`
}

// CreateExternalTransfer handles the creation of an external bank transfer
func (server *Server) CreateExternalTransfer(ctx *gin.Context) {
	var req externalTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if the account belongs to the authenticated user
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetAccount(ctx, req.FromAccountID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("account not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if account.Owner != authPayload.Username {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("account doesn't belong to the authenticated user")))
		return
	}

	// Check if the account currency matches the transfer currency
	if account.Currency != req.Currency {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("account currency mismatch: %s vs %s", account.Currency, req.Currency)))
		return
	}

	// Verify the account has enough balance
	if account.Balance < req.Amount {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("insufficient funds")))
		return
	}

	// Validate the recipient account
	bankAPI, err := server.bankAPIProvider.GetBankAPI(req.ToBankCode)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("bank with code %s is not supported", req.ToBankCode)))
		return
	}

	recipientName, err := bankAPI.ValidateAccount(ctx, req.ToBankCode, req.ToAccountNumber)
	if err != nil {
		if errors.Is(err, bankapi.ErrAccountNotFound) {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("recipient account not found at the bank")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to validate recipient account")))
		return
	}

	// Generate a unique reference
	reference := fmt.Sprintf("NEXT-%s", uuid.New().String()[:8])

	// Create external transfer record
	arg := db.CreateExternalTransferParams{
		FromAccountID:   req.FromAccountID,
		ToBankCode:      req.ToBankCode,
		ToAccountNumber: req.ToAccountNumber,
		RecipientName:   recipientName,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Reference:       reference,
		Description:     sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	// Start a transaction
	transfer, err := server.store.CreateExternalTransfer(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Update status to processing
	updateArg := db.UpdateExternalTransferStatusParams{
		ID:     transfer.ID,
		Status: "processing",
	}
	transfer, err = server.store.UpdateExternalTransferStatus(ctx, updateArg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Create transfer request
	description := ""
	if transfer.Description.Valid {
		description = transfer.Description.String
	}

	transferReq := bankapi.TransferRequest{
		Amount:            transfer.Amount,
		Currency:          transfer.Currency,
		FromAccountNumber: fmt.Sprintf("%d", account.ID), // Convert to string
		ToAccountNumber:   transfer.ToAccountNumber,
		ToBankCode:        transfer.ToBankCode,
		RecipientName:     transfer.RecipientName,
		Reference:         transfer.Reference,
		Description:       description,
	}

	// Send transfer request to bank
	transferResp, err := bankAPI.TransferMoney(ctx, transferReq)
	if err != nil {
		// Update status to failed and return the error
		errorMsg := "Failed to send money to the bank"
		if errors.Is(err, bankapi.ErrInsufficientFunds) {
			errorMsg = "Insufficient funds at the bank"
		} else if errors.Is(err, bankapi.ErrBankAPINotAvailable) {
			errorMsg = "Bank API is temporarily unavailable"
		} else if errors.Is(err, bankapi.ErrTransferFailed) {
			errorMsg = "Bank transfer failed"
		}

		updateArg := db.UpdateExternalTransferStatusParams{
			ID:           transfer.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: errorMsg, Valid: true},
		}
		transfer, _ = server.store.UpdateExternalTransferStatus(ctx, updateArg)

		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New(errorMsg)))
		return
	}

	// Deduct money from sender account
	balanceArg := db.AddAccountBalanceParams{
		ID:     account.ID,
		Amount: -transfer.Amount,
	}
	_, err = server.store.AddAccountBalance(ctx, balanceArg)
	if err != nil {
		// Update status to failed and return the error
		updateArg := db.UpdateExternalTransferStatusParams{
			ID:           transfer.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: "Failed to deduct balance", Valid: true},
		}
		transfer, _ = server.store.UpdateExternalTransferStatus(ctx, updateArg)

		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("Failed to deduct balance")))
		return
	}

	// Create an entry for the deduction
	_, err = server.store.CreateEntry(ctx, db.CreateEntryParams{
		AccountID: account.ID,
		Amount:    -transfer.Amount,
	})
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create entry for external transfer %d", transfer.ID)
		// This is not critical - the money has been sent and deducted
	}

	// Update transfer status to completed
	updateCompleteArg := db.UpdateExternalTransferStatusParams{
		ID:              transfer.ID,
		Status:          "completed",
		TransactionID:   sql.NullString{String: transferResp.TransactionID, Valid: transferResp.TransactionID != ""},
		TransactionFees: sql.NullInt64{Int64: transferResp.TransactionFees, Valid: transferResp.TransactionFees > 0},
	}
	transfer, err = server.store.UpdateExternalTransferStatus(ctx, updateCompleteArg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := externalTransferResponse{
		ID:              transfer.ID,
		FromAccountID:   transfer.FromAccountID,
		ToBankCode:      transfer.ToBankCode,
		ToAccountNumber: transfer.ToAccountNumber,
		RecipientName:   transfer.RecipientName,
		Amount:          transfer.Amount,
		Currency:        transfer.Currency,
		Status:          transfer.Status,
		Reference:       transfer.Reference,
		CreatedAt:       transfer.CreatedAt,
	}

	// Add additional fields if available
	if transfer.Description.Valid {
		response.Description = transfer.Description.String
	}

	if transfer.TransactionID.Valid {
		response.TransactionID = transfer.TransactionID.String
	}

	if transfer.TransactionFees.Valid {
		response.TransactionFees = transfer.TransactionFees.Int64
	}

	if transfer.ErrorMessage.Valid {
		response.ErrorMessage = transfer.ErrorMessage.String
	}

	ctx.JSON(http.StatusOK, response)
}

// processExternalTransfer processes an external transfer request asynchronously
// This is kept for background processing if needed
func (server *Server) processExternalTransfer(ctx context.Context, transferID int64) {
	// Get the transfer details
	transfer, err := server.store.GetExternalTransfer(ctx, transferID)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get external transfer %d", transferID)
		return
	}

	// Update status to processing
	updateArg := db.UpdateExternalTransferStatusParams{
		ID:     transferID,
		Status: "processing",
	}
	transfer, err = server.store.UpdateExternalTransferStatus(ctx, updateArg)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to update external transfer status %d", transferID)
		return
	}

	// Get the account details
	account, err := server.store.GetAccount(ctx, transfer.FromAccountID)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get account for external transfer %d", transferID)
		updateErrorStatus(ctx, server.store, transferID, "failed", "Account not found")
		return
	}

	// Verify the account has enough balance
	if account.Balance < transfer.Amount {
		logrus.Errorf("Insufficient funds for external transfer %d", transferID)
		updateErrorStatus(ctx, server.store, transferID, "failed", "Insufficient funds")
		return
	}

	// Get the bank API
	bankAPI, err := server.bankAPIProvider.GetBankAPI(transfer.ToBankCode)
	if err != nil {
		logrus.WithError(err).Errorf("Bank API not available for external transfer %d", transferID)
		updateErrorStatus(ctx, server.store, transferID, "failed", "Bank API not available")
		return
	}

	// Create transfer request
	description := ""
	if transfer.Description.Valid {
		description = transfer.Description.String
	}

	transferReq := bankapi.TransferRequest{
		Amount:            transfer.Amount,
		Currency:          transfer.Currency,
		FromAccountNumber: fmt.Sprintf("%d", account.ID), // Convert to string
		ToAccountNumber:   transfer.ToAccountNumber,
		ToBankCode:        transfer.ToBankCode,
		RecipientName:     transfer.RecipientName,
		Reference:         transfer.Reference,
		Description:       description,
	}

	// Send transfer request to bank
	transferResp, err := bankAPI.TransferMoney(ctx, transferReq)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to send money for external transfer %d", transferID)
		errMsg := "Failed to send money to the bank"
		if errors.Is(err, bankapi.ErrInsufficientFunds) {
			errMsg = "Insufficient funds at the bank"
		} else if errors.Is(err, bankapi.ErrBankAPINotAvailable) {
			errMsg = "Bank API is temporarily unavailable"
		} else if errors.Is(err, bankapi.ErrTransferFailed) {
			errMsg = "Bank transfer failed"
		}
		updateErrorStatus(ctx, server.store, transferID, "failed", errMsg)
		return
	}

	// Deduct money from sender account
	arg := db.AddAccountBalanceParams{
		ID:     account.ID,
		Amount: -transfer.Amount,
	}
	_, err = server.store.AddAccountBalance(ctx, arg)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to deduct balance for external transfer %d", transferID)
		updateErrorStatus(ctx, server.store, transferID, "failed", "Failed to deduct balance")
		return
	}

	// Create an entry for the deduction
	_, err = server.store.CreateEntry(ctx, db.CreateEntryParams{
		AccountID: account.ID,
		Amount:    -transfer.Amount,
	})
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create entry for external transfer %d", transferID)
		// This is not critical - the money has been sent and deducted
	}

	// Update transfer status to completed
	updateCompleteArg := db.UpdateExternalTransferStatusParams{
		ID:              transferID,
		Status:          "completed",
		TransactionID:   sql.NullString{String: transferResp.TransactionID, Valid: transferResp.TransactionID != ""},
		TransactionFees: sql.NullInt64{Int64: transferResp.TransactionFees, Valid: transferResp.TransactionFees > 0},
	}
	_, err = server.store.UpdateExternalTransferStatus(ctx, updateCompleteArg)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to update external transfer status to completed %d", transferID)
		// Not critical - the transfer has been completed
	}

	logrus.Infof("External transfer %d completed successfully", transferID)
}

// updateErrorStatus updates the status of an external transfer to failed with the given error message
func updateErrorStatus(ctx context.Context, store db.Store, transferID int64, status, errorMsg string) {
	updateArg := db.UpdateExternalTransferStatusParams{
		ID:           transferID,
		Status:       status,
		ErrorMessage: sql.NullString{String: errorMsg, Valid: true},
	}
	_, err := store.UpdateExternalTransferStatus(ctx, updateArg)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to update external transfer status to %s: %d", status, transferID)
	}
}

// GetExternalTransfer retrieves an external transfer by ID
func (server *Server) GetExternalTransfer(ctx *gin.Context) {
	var req struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	transfer, err := server.store.GetExternalTransfer(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("external transfer not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if the transfer belongs to the authenticated user
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetAccount(ctx, transfer.FromAccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if account.Owner != authPayload.Username {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("transfer doesn't belong to the authenticated user")))
		return
	}

	response := externalTransferToResponse(transfer)
	ctx.JSON(http.StatusOK, response)
}

// ListExternalTransfers lists all external transfers for a user's account
func (server *Server) ListExternalTransfers(ctx *gin.Context) {
	var req struct {
		AccountID int64 `form:"account_id" binding:"required,min=1"`
		PageID    int32 `form:"page_id" binding:"required,min=1"`
		PageSize  int32 `form:"page_size" binding:"required,min=5,max=10"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if the account belongs to the authenticated user
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("account not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if account.Owner != authPayload.Username {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("account doesn't belong to the authenticated user")))
		return
	}

	arg := db.ListExternalTransfersParams{
		FromAccountID: req.AccountID,
		Limit:         req.PageSize,
		Offset:        (req.PageID - 1) * req.PageSize,
	}

	transfers, err := server.store.ListExternalTransfers(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	responses := make([]externalTransferResponse, len(transfers))
	for i, transfer := range transfers {
		responses[i] = externalTransferToResponse(transfer)
	}

	ctx.JSON(http.StatusOK, responses)
}

// Helper function to convert db.ExternalTransfer to externalTransferResponse
func externalTransferToResponse(transfer db.ExternalTransfer) externalTransferResponse {
	resp := externalTransferResponse{
		ID:              transfer.ID,
		FromAccountID:   transfer.FromAccountID,
		ToBankCode:      transfer.ToBankCode,
		ToAccountNumber: transfer.ToAccountNumber,
		RecipientName:   transfer.RecipientName,
		Amount:          transfer.Amount,
		Currency:        transfer.Currency,
		Status:          transfer.Status,
		Reference:       transfer.Reference,
		CreatedAt:       transfer.CreatedAt,
	}

	if transfer.Description.Valid {
		resp.Description = transfer.Description.String
	}

	if transfer.TransactionID.Valid {
		resp.TransactionID = transfer.TransactionID.String
	}

	if transfer.TransactionFees.Valid {
		resp.TransactionFees = transfer.TransactionFees.Int64
	}

	if transfer.ErrorMessage.Valid {
		resp.ErrorMessage = transfer.ErrorMessage.String
	}

	return resp
}

// LookupBankAccount validates a bank account number and returns the account holder name
func (server *Server) LookupBankAccount(ctx *gin.Context) {
	var req accountLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Log the bank account lookup attempt
	logrus.WithFields(logrus.Fields{
		"bank_code":      req.BankCode,
		"account_number": req.AccountNumber,
	}).Info("Bank account lookup request received")

	// Get bank API for the requested bank code
	bankAPI, err := server.bankAPIProvider.GetBankAPI(req.BankCode)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("bank with code %s is not supported", req.BankCode)))
		return
	}

	// Validate the account
	accountName, err := bankAPI.ValidateAccount(ctx, req.BankCode, req.AccountNumber)

	// Prepare the response with additional demo-friendly information
	response := accountLookupResponse{
		BankCode:      req.BankCode,
		AccountNumber: req.AccountNumber,
		Found:         err == nil,
	}

	if err == nil {
		response.AccountName = accountName
		logrus.WithFields(logrus.Fields{
			"bank_code":      req.BankCode,
			"account_number": req.AccountNumber,
			"account_name":   accountName,
		}).Info("Bank account found")
	} else {
		logrus.WithFields(logrus.Fields{
			"bank_code":      req.BankCode,
			"account_number": req.AccountNumber,
			"error":          err.Error(),
		}).Info("Bank account not found")
	}

	ctx.JSON(http.StatusOK, response)
}
