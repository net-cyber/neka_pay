package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/util/sms"
	"github.com/sirupsen/logrus"
)

type sendVerificationCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type sendVerificationCodeResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PhoneNumber string `json:"phone_number"`
	ExpiresAt   string `json:"expires_at"`
}

type verifyPhoneNumberRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	OTP         string `json:"otp" binding:"required,len=6"`
}

// verifyPhoneNumberResponse is the response for phone verification
type verifyPhoneNumberResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	PhoneNumber   string `json:"phone_number"`
	PhoneVerified bool   `json:"phone_verified"`
}

// sendVerificationCode sends an OTP to a phone number
func (server *Server) sendVerificationCode(ctx *gin.Context) {
	var req sendVerificationCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind JSON for sendVerificationCode")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if user exists with this phone number
	user, err := server.store.GetUserByPhone(ctx, req.PhoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			errMsg := errors.New("no user found with this phone number")
			ctx.JSON(http.StatusNotFound, errorResponse(errMsg))
			return
		}
		logrus.WithError(err).Error("Failed to get user by phone")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// If user already verified, return a message
	if user.PhoneVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Phone number already verified",
		})
		return
	}
	// Check if there were too many recent attempts
	recentAttempts, err := server.store.CountRecentOTPAttempts(ctx, db.CountRecentOTPAttemptsParams{
		PhoneNumber: req.PhoneNumber,
		CreatedAt:   time.Now().Add(-1 * time.Hour), // Check attempts in the last hour
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to count recent OTP attempts")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if recentAttempts > 5 { // Limit to 5 attempts per hour
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "Too many verification attempts. Please try again later.",
		})
		return
	}

	// Generate OTP
	otp := sms.GenerateOTP()
	expiresAt := time.Now().Add(server.config.OTPExpiryDuration)

	// Save OTP to database
	_, err = server.store.CreateOTPVerification(ctx, db.CreateOTPVerificationParams{
		PhoneNumber: req.PhoneNumber,
		Otp:         otp,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to create OTP verification record")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Create SMS client
	afroSMS := sms.NewAfroSMS(server.config.AfroSMSToken, server.config.AfroSMSIdentifierID)

	// Send OTP via SMS
	_, err = afroSMS.SendVerificationSMS(req.PhoneNumber, otp)
	if err != nil {
		logrus.WithError(err).Error("Failed to send SMS verification code")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	logrus.WithFields(logrus.Fields{
		"phone_number": req.PhoneNumber,
	}).Info("Verification code sent successfully")

	rsp := sendVerificationCodeResponse{
		Success:     true,
		Message:     "Verification code sent successfully",
		PhoneNumber: req.PhoneNumber,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}
	ctx.JSON(http.StatusOK, rsp)
}

// verifyPhoneNumber verifies a phone number with an OTP
func (server *Server) verifyPhoneNumber(ctx *gin.Context) {
	var req verifyPhoneNumberRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind JSON for verifyPhoneNumber")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if user exists with this phone number
	user, err := server.store.GetUserByPhone(ctx, req.PhoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			errMsg := errors.New("no user found with this phone number")
			ctx.JSON(http.StatusNotFound, errorResponse(errMsg))
			return
		}
		logrus.WithError(err).Error("Failed to get user by phone")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// If user already verified, return a message
	if user.PhoneVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Phone number already verified",
		})
		return
	}

	// Get the latest OTP verification record
	otpVerification, err := server.store.GetLatestOTPVerification(ctx, req.PhoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			errMsg := errors.New("no verification code found or code expired")
			ctx.JSON(http.StatusNotFound, errorResponse(errMsg))
			return
		}
		logrus.WithError(err).Error("Failed to get OTP verification record")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if OTP is expired
	if time.Now().After(otpVerification.ExpiresAt) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Verification code expired. Please request a new one.",
		})
		return
	}

	// Check if OTP is already verified
	if otpVerification.Verified {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Verification code already used",
		})
		return
	}

	// Validate OTP
	if otpVerification.Otp != req.OTP {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid verification code",
		})
		return
	}

	// Mark OTP as verified
	_, err = server.store.MarkOTPAsVerified(ctx, otpVerification.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to mark OTP as verified")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Update user's phone_verified status
	err = server.store.UpdateUserPhoneVerified(ctx, req.PhoneNumber)
	if err != nil {
		logrus.WithError(err).Error("Failed to update user's phone verified status")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	logrus.WithFields(logrus.Fields{
		"phone_number": req.PhoneNumber,
		"username":     user.Username,
	}).Info("Phone number verified successfully")

	rsp := verifyPhoneNumberResponse{
		Success:       true,
		Message:       "Phone number verified successfully",
		PhoneNumber:   req.PhoneNumber,
		PhoneVerified: true,
	}
	ctx.JSON(http.StatusOK, rsp)
}

// resendVerificationCode resends an OTP to a phone number
func (server *Server) resendVerificationCode(ctx *gin.Context) {
	var req sendVerificationCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind JSON for resendVerificationCode")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Check if user exists with this phone number
	user, err := server.store.GetUserByPhone(ctx, req.PhoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			errMsg := errors.New("no user found with this phone number")
			ctx.JSON(http.StatusNotFound, errorResponse(errMsg))
			return
		}
		logrus.WithError(err).Error("Failed to get user by phone")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// If user already verified, return a message
	if user.PhoneVerified {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Phone number already verified",
		})
		return
	}

	// Check if there were too many recent attempts
	recentAttempts, err := server.store.CountRecentOTPAttempts(ctx, db.CountRecentOTPAttemptsParams{
		PhoneNumber: req.PhoneNumber,
		CreatedAt:   time.Now().Add(-1 * time.Hour), // Check attempts in the last hour
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to count recent OTP attempts")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if recentAttempts > 5 { // Limit to 5 attempts per hour
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "Too many verification attempts. Please try again later.",
		})
		return
	}

	// Generate new OTP
	otp := sms.GenerateOTP()
	expiresAt := time.Now().Add(server.config.OTPExpiryDuration)

	// Save OTP to database
	_, err = server.store.CreateOTPVerification(ctx, db.CreateOTPVerificationParams{
		PhoneNumber: req.PhoneNumber,
		Otp:         otp,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to create OTP verification record")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Create SMS client
	afroSMS := sms.NewAfroSMS(server.config.AfroSMSToken, server.config.AfroSMSIdentifierID)

	// Send OTP via SMS
	_, err = afroSMS.SendVerificationSMS(req.PhoneNumber, otp)
	if err != nil {
		logrus.WithError(err).Error("Failed to send SMS verification code")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Invalidate previous OTPs
	err = server.store.InvalidatePreviousOTPs(ctx, req.PhoneNumber)
	if err != nil {
		logrus.WithError(err).Error("Failed to invalidate previous OTPs")
		// Continue anyway, as this is not critical
	}

	logrus.WithFields(logrus.Fields{
		"phone_number": req.PhoneNumber,
	}).Info("Verification code resent successfully")

	rsp := sendVerificationCodeResponse{
		Success:     true,
		Message:     "Verification code resent successfully",
		PhoneNumber: req.PhoneNumber,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}
	ctx.JSON(http.StatusOK, rsp)
}
