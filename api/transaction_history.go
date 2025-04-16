package api

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/token"
	"github.com/sirupsen/logrus"
)

// TransactionType represents the various types of transactions
type TransactionType string

const (
	TransactionTypeInternalTransfer                 = "internal_transfer"
	TransactionTypeExternalTransfer                 = "external_transfer"
	TransactionTypeTopUp                            = "top_up"
	TransactionTypeUnknown          TransactionType = "unknown"
)

// TransactionDirection indicates whether money was sent or received
type TransactionDirection string

const (
	TransactionDirectionIncoming = "incoming"
	TransactionDirectionOutgoing = "outgoing"
)

// TransactionHistoryResponse is a unified format for all transaction types
type TransactionHistoryResponse struct {
	ID               int64                `json:"id"`
	Type             TransactionType      `json:"type"`
	Direction        TransactionDirection `json:"direction"`
	Amount           int64                `json:"amount"`
	Currency         string               `json:"currency"`
	Status           string               `json:"status"`
	Reference        string               `json:"reference,omitempty"`
	Description      string               `json:"description,omitempty"`
	CounterpartyID   int64                `json:"counterparty_id,omitempty"`
	CounterpartyName string               `json:"counterparty_name,omitempty"`
	BankCode         string               `json:"bank_code,omitempty"`
	AccountNumber    string               `json:"account_number,omitempty"`
	TransactionFees  int64                `json:"transaction_fees,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
}

// TransactionListRequest holds the parameters for listing transactions
type TransactionListRequest struct {
	AccountID int64 `form:"account_id" binding:"required,min=1"`
	PageID    int32 `form:"page_id" binding:"required,min=1"`
	PageSize  int32 `form:"page_size" binding:"required,min=5,max=20"`
	// Optional filter fields could be added here
}

// GetTransactionHistory retrieves the consolidated transaction history for an account
func (server *Server) GetTransactionHistory(ctx *gin.Context) {
	var req TransactionListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default page size if not provided
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// Set default page ID if not provided
	if req.PageID == 0 {
		req.PageID = 1
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Verify account ownership
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if the authenticated user owns the account
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	transactions, err := server.getUnifiedTransactions(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transactions)
}

// getUnifiedTransactions retrieves and combines transactions from multiple sources
func (server *Server) getUnifiedTransactions(ctx *gin.Context, req TransactionListRequest) ([]TransactionHistoryResponse, error) {
	var allTransactions []TransactionHistoryResponse

	// Get account to know the currency
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}

	// 1. Get internal transfers (both sent and received)
	transferParams := db.ListTransfersParams{
		FromAccountID: req.AccountID,
		Limit:         req.PageSize,
		Offset:        (req.PageID - 1) * req.PageSize,
	}

	transfers, err := server.store.ListTransfers(ctx, transferParams)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve internal transfers")
		return nil, err
	}

	// Process internal transfers
	for _, transfer := range transfers {
		var direction TransactionDirection
		var counterpartyID int64
		var status string = "completed" // Internal transfers are always completed

		if transfer.FromAccountID == req.AccountID {
			direction = TransactionDirectionOutgoing
			counterpartyID = transfer.ToAccountID
		} else {
			direction = TransactionDirectionIncoming
			counterpartyID = transfer.FromAccountID
		}

		// Get counterparty name
		counterpartyName := ""
		counterparty, err := server.store.GetAccount(ctx, counterpartyID)
		if err == nil {
			// Get user's full name
			user, err := server.store.GetUser(ctx, counterparty.Owner)
			if err == nil {
				counterpartyName = user.FullName
			}
		}

		transaction := TransactionHistoryResponse{
			ID:               transfer.ID,
			Type:             TransactionTypeInternalTransfer,
			Direction:        direction,
			Amount:           transfer.Amount,
			Currency:         account.Currency, // Using the account's currency
			Status:           status,
			CounterpartyID:   counterpartyID,
			CounterpartyName: counterpartyName,
			CreatedAt:        transfer.CreatedAt,
		}

		allTransactions = append(allTransactions, transaction)
	}

	// 2. Get external transfers (only outgoing for now)
	externalParams := db.ListExternalTransfersParams{
		FromAccountID: req.AccountID,
		Limit:         req.PageSize,
		Offset:        (req.PageID - 1) * req.PageSize,
	}

	externalTransfers, err := server.store.ListExternalTransfers(ctx, externalParams)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve external transfers")
		return nil, err
	}

	// Process external transfers
	for _, transfer := range externalTransfers {
		// External transfers are always outgoing
		transaction := TransactionHistoryResponse{
			ID:               transfer.ID,
			Type:             TransactionTypeExternalTransfer,
			Direction:        TransactionDirectionOutgoing,
			Amount:           transfer.Amount,
			Currency:         transfer.Currency,
			Status:           transfer.Status,
			Reference:        transfer.Reference,
			BankCode:         transfer.ToBankCode,
			AccountNumber:    transfer.ToAccountNumber,
			CounterpartyName: transfer.RecipientName,
			CreatedAt:        transfer.CreatedAt,
		}

		if transfer.TransactionFees.Valid {
			transaction.TransactionFees = transfer.TransactionFees.Int64
		}

		if transfer.Description.Valid {
			transaction.Description = transfer.Description.String
		}

		allTransactions = append(allTransactions, transaction)
	}

	// 3. Add entries for top-ups and other transactions not covered above
	entriesParams := db.ListEntriesParams{
		AccountID: req.AccountID,
		Limit:     req.PageSize,
		Offset:    (req.PageID - 1) * req.PageSize,
	}

	entries, err := server.store.ListEntries(ctx, entriesParams)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve entries")
		return nil, err
	}

	// Track which entries are already covered by transfers to avoid duplication
	processedEntries := make(map[int64]bool)

	// Process entries
	for _, entry := range entries {
		// Skip entries that are part of internal/external transfers
		// In a real application, you would need a way to link entries to transfers
		// This is a simplified approach
		if processedEntries[entry.ID] {
			continue
		}

		// If amount is positive, it's likely a top-up
		if entry.Amount > 0 {
			transaction := TransactionHistoryResponse{
				ID:          entry.ID,
				Type:        TransactionTypeTopUp,
				Direction:   TransactionDirectionIncoming,
				Amount:      entry.Amount,
				Currency:    account.Currency,
				Status:      "completed",
				Description: "Account top-up",
				CreatedAt:   entry.CreatedAt,
			}
			allTransactions = append(allTransactions, transaction)
		}
	}

	// Sort by date (newest first)
	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].CreatedAt.After(allTransactions[j].CreatedAt)
	})

	return allTransactions, nil
}
