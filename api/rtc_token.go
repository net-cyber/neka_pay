package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/net-cyber/neka_pay/token"
)

// GetRTCTokenRequest defines the request structure for obtaining an RTC token
type GetRTCTokenRequest struct {
	ChannelName string `json:"channel_name" form:"channel_name" binding:"required"`
	UID         uint32 `json:"uid" form:"uid" binding:"omitempty"`
	Role        uint32 `json:"role" form:"role" binding:"omitempty"` // 1 for publisher, 2 for subscriber
}

// GetRTCTokenResponse defines the response structure for the RTC token
type GetRTCTokenResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Msg  string `json:"msg"`
}

// getRTCToken handles the generation of Agora RTC tokens
// This endpoint is now available for non-authorized users
func (server *Server) getRTCToken(ctx *gin.Context) {
	var req GetRTCTokenRequest

	// Try to bind from both JSON and form data to match PHP's behavior
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, GetRTCTokenResponse{
			Code: -1,
			Data: "",
			Msg:  err.Error(),
		})
		return
	}

	if server.rtcTokenMaker == nil {
		ctx.JSON(http.StatusInternalServerError, GetRTCTokenResponse{
			Code: -1,
			Data: "",
			Msg:  "token error: RTC token service is not configured",
		})
		return
	}

	// Default to 0 if UID is not provided
	uid := req.UID

	// Default to publisher role (1) if not specified
	role := token.RolePublisher
	if req.Role == 2 {
		role = token.RoleSubscriber
	}

	// Generate RTC token with configured duration
	tokenDuration := server.config.AgoraRTCTokenDuration
	if tokenDuration == 0 {
		// Default to 2 hours if not configured
		tokenDuration = 2 * time.Hour
	}

	// Convert duration to seconds for token expiration
	expireSeconds := uint32(tokenDuration.Seconds())

	// Use BuildTokenWithUid to support role-based permissions
	token, err := server.rtcTokenMaker.BuildTokenWithUid(req.ChannelName, uid, role, expireSeconds, expireSeconds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, GetRTCTokenResponse{
			Code: -1,
			Data: "",
			Msg:  "token error: " + err.Error(),
		})
		return
	}

	response := GetRTCTokenResponse{
		Code: 0,
		Data: token,
		Msg:  "success",
	}

	ctx.JSON(http.StatusOK, response)
}
