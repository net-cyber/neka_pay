package api

import (
	// "errors"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/net-cyber/neka_pay/db/sqlc"

	// "github.com/google/uuid"
	"github.com/net-cyber/neka_pay/util"
	"github.com/sirupsen/logrus"
)

type createUserRequest struct {
	Username                   string `json:"username" binding:"required,alphanum"`
	Password                   string `json:"password" binding:"required,min=6"`
	FullName                   string `json:"full_name" binding:"required"`
	International_phone_number string `json:"international_phone_number" binding:"required"`
}

type userResponse struct {
	Username                   string    `json:"username"`
	FullName                   string    `json:"full_name"`
	International_phone_number string    `json:"international_phone_number"`
	PhoneVerified              bool      `json:"phone_verified"`
	PasswordChangedAt          time.Time `json:"password_changed_at"`
	CreatedAt                  time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:                   user.Username,
		FullName:                   user.FullName,
		International_phone_number: user.InternationalPhoneNumber,
		PhoneVerified:              user.PhoneVerified,
		PasswordChangedAt:          user.PasswordChangedAt,
		CreatedAt:                  user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind JSON for createUser")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		logrus.WithError(err).Error("Failed to hash password")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:                 req.Username,
		HashedPassword:           hashedPassword,
		FullName:                 req.FullName,
		InternationalPhoneNumber: req.International_phone_number,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		logrus.WithError(err).Error("Failed to create user in database")
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	logrus.WithField("username", user.Username).Info("User created successfully")
	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind JSON for loginUser")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user from database")
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		logrus.WithError(err).Error("Password check failed")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to create access token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		accessPayload.Role,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to create session")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	logrus.WithField("username", user.Username).Info("User logged in successfully")
	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, rsp)
}
