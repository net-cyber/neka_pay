package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
)

type transferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=0"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=0"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (server *Server) CreateTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if !server.validAccount(ctx, req.FromAccountID, req.Currency) {
		return
	}

	if !server.validAccount(ctx, req.ToAccountID, req.Currency) {
		return
	}
	if server.isSelfSend(ctx, req.FromAccountID, req.ToAccountID) {
		return
	}
	if !server.senderHasBalance(ctx, req.FromAccountID, req.Amount) {
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	transferResult, err := server.store.TransferTx(ctx, arg)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transferResult)
}
func (server *Server) senderHasBalance(ctx *gin.Context, senderAccountID int64, amount int64) bool {
	account, err := server.store.GetAccount(ctx, senderAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return false
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return false
	}

	if account.Balance <= amount  {
		err := fmt.Errorf("dont have sufficent balance to make this transaction")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}
	
	return true
}
func (server *Server) isSelfSend(ctx *gin.Context, fromAccountID int64, toAccountID int64) bool {
	if fromAccountID == toAccountID {
		err := fmt.Errorf("sending to yourself is not permitted")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return true
	}
	return false
}
func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) bool {
	account, err := server.store.GetAccount(ctx, accountID)

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return false
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return false
	}
	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}
	return true
}
