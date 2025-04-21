package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/token"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

type listUsersRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type listUsersResponse struct {
	Username                   string `json:"username"`
	FullName                   string `json:"full_name"`
	International_phone_number string `json:"international_phone_number"`
	Avatar                     string `json:"avatar"`
	Fcmtoken                   string `json:"fcmtoken"`
	Online                     bool   `json:"online"`
	Token                      string `json:"token"`
	Role                       string `json:"role"`
	PhoneVerified              bool   `json:"phone_verified"`
}

func newListUsersResponse(user db.User) listUsersResponse {
	return listUsersResponse{
		Username:                   user.Username,
		FullName:                   user.FullName,
		International_phone_number: user.InternationalPhoneNumber,
		Avatar:                     user.Avatar,
		PhoneVerified:              user.PhoneVerified,
		Fcmtoken:                   user.Fcmtoken,
		Online:                     user.Online,
		Token:                      user.Token,
		Role:                       user.Role,
	}
}

func (server *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	arg := db.ListUsersOthersParams{
		Limit:    req.PageSize,
		Offset:   (req.PageID - 1) * req.PageSize,
		Username: authPayload.Username,
	}

	users, err := server.store.ListUsersOthers(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]listUsersResponse, len(users))
	for i, user := range users {

		response[i] = newListUsersResponse(user)
	}

	ctx.JSON(http.StatusOK, response)
}

type updateUserFCMTokenRequest struct {
	Fcmtoken string `json:"fcmtoken" binding:"required"`
}
type updateUserFCMTokenResponse struct {
	Message string `json:"message"`
}

func (server *Server) updateUserFCMToken(ctx *gin.Context) {
	var req updateUserFCMTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.UpdateUserFCMTokenParams{
		Username: authPayload.Username,
		Fcmtoken: req.Fcmtoken,
	}
	err := server.store.UpdateUserFCMToken(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "FCM token updated successfully"})
}

func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		errCode := db.ErrorCode(err)
		if errCode == db.ForeignKeyViolation || errCode == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type getAccountForVerificationRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type getAccountForVerificationResponse struct {
	FullName string `json:"full_name"`
}

func (server *Server) getAccountForVerification(ctx *gin.Context) {
	var req getAccountForVerificationRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner == authPayload.Username {
		err := errors.New("You cannot send to yourself")
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, account.Owner)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, getAccountForVerificationResponse{
		FullName: user.FullName,
	})
}

func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type validateUserAccountBalanceRequest struct {
	Amount    int64 `json:"amount" binding:"required,gt=0"`
	AccountID int64 `json:"account_id" binding:"required,min=1"`
}

func (server *Server) validateUserAccountBalance(ctx *gin.Context) {
	var req validateUserAccountBalanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	// check if account exists
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// check if account belongs to the authenticated user
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// check if account balance is sufficient
	if account.Balance < req.Amount {
		err := errors.New("account balance is less than the amount")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	// check if remaining balance is less than 50
	if account.Balance < (req.Amount + 50) {
		err := errors.New("Remaining balance should be more than 50")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account balance is sufficient",
	})
}

type listAccountRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}

type topUpAccountRequest struct {
	Amount   int64  `json:"amount" binding:"required,gt=0"`
	Currency string `json:"currency" binding:"required,currency"`
}

func (server *Server) topUpAccount(ctx *gin.Context) {
	var req topUpAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var idParam getAccountRequest
	if err := ctx.ShouldBindUri(&idParam); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get account to verify ownership and currency match
	account, err := server.store.GetAccount(ctx, idParam.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify account ownership
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Verify currency match
	if account.Currency != req.Currency {
		err := fmt.Errorf("account currency mismatch: %s vs %s", account.Currency, req.Currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Perform top-up transaction
	arg := db.TopUpTxParams{
		AccountID: idParam.ID,
		Amount:    req.Amount,
	}

	result, err := server.store.TopUpTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Return both the updated account and the entry record
	response := gin.H{
		"account": result.Account,
		"entry":   result.Entry,
	}

	ctx.JSON(http.StatusOK, response)
}
