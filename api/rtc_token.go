package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetRTCTokenRequest defines the request structure for obtaining an RTC token
type GetRTCTokenRequest struct {
	ChannelName string `json:"channel_name" binding:"required"`
	UID         uint32 `json:"uid" binding:"omitempty"`
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
	if err := ctx.ShouldBindJSON(&req); err != nil {
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

	// Generate RTC token with configured duration
	tokenDuration := server.config.AgoraRTCTokenDuration
	if tokenDuration == 0 {
		// Default to 2 hours if not configured
		tokenDuration = 2 * time.Hour
	}

	token, err := server.rtcTokenMaker.CreateRTCToken(req.ChannelName, uid, tokenDuration)
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
