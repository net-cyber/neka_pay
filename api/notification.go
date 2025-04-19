package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"google.golang.org/api/option"
)

type sendNoticeRequest struct {
	ToToken    string `json:"to_token" binding:"required"`
	ToName     string `json:"to_name"`
	ToAvatar   string `json:"to_avatar"`
	CallType   string `json:"call_type" binding:"required,oneof=voice video text cancel"`
}

type noticeResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func (server *Server) sendNotice(ctx *gin.Context) {
	var req sendNoticeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON request: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// log.Printf("Notification request received: type=%s, from=%s to=%s", req.CallType, req.UserName, req.ToToken)

	// Find user by token
	user, err := server.store.GetUserByToken(ctx, req.ToToken)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			log.Printf("User not found with token: %s", req.ToToken)
			ctx.JSON(http.StatusOK, errorResponse(err))
			return
		}
		log.Printf("Database error when finding user: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	deviceToken := user.Fcmtoken
	if deviceToken == "" {
		log.Printf("User found but has empty FCM token: username=%s", user.Username)
		ctx.JSON(http.StatusOK, errorResponse(err))
		return
	}

	log.Printf("User found with valid FCM token: username=%s", user.Username)

	// Initialize Firebase app
	opt := option.WithCredentialsFile("supper_app_notification.json")
	log.Println("Initializing Firebase app...")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("Error initializing Firebase app: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get Firebase messaging client
	log.Println("Getting Firebase messaging client...")
	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("Error getting Firebase messaging client: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Prepare message data (common for all message types)
	data := map[string]string{
		"token":     user.Token,
		"avatar":    user.Avatar,
		"name":      user.Username,
		"call_type": req.CallType,
	}

	var message *messaging.Message
	log.Printf("Preparing %s notification message...", req.CallType)

	switch req.CallType {
	case "cancel":
		message = &messaging.Message{
			Token: deviceToken,
			Data:  data,
		}
	case "voice":
		message = &messaging.Message{
			Token: deviceToken,
			Data:  data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "com.dbestech.chatty.call",
					Title:     "Voice call made by " + user.Username,
					Body:      "Please click to answer the voice call",
				},
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{
					"apns-priority": "10",
				},
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: "Voice call made by " + user.Username,
							Body:  "Please click to answer the voice call",
						},
						Badge: func() *int { i := 1; return &i }(),
						Sound: "task_cancel.caf",
					},
				},
			},
		}
	case "video":
		message = &messaging.Message{
			Token: deviceToken,
			Data:  data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "com.dbestech.chatty.call",
					Title:     "Video call made by " + user.Username,
					Body:      "Please click to answer the video call",
				},
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{
					"apns-priority": "10",
				},
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: "Video call made by " + user.Username,
							Body:  "Please click to answer the video call",
						},
						Badge: func() *int { i := 1; return &i }(),
						Sound: "task_cancel.caf",
					},
				},
			},
		}
	case "text":
		message = &messaging.Message{
			Token: deviceToken,
			Data:  data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "com.dbestech.chatty.message",
					Title:     "Message made by " + user.Username,
					Body:      "Please click to answer the Message",
				},
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{
					"apns-priority": "10",
				},
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: "Message made by " + user.Username,
							Body:  "Please click to answer the Message",
						},
						Badge: func() *int { i := 1; return &i }(),
						Sound: "ding.caf",
					},
				},
			},
		}
	}

	// Send the message
	log.Printf("Sending %s notification to user: %s (device token: %s)", req.CallType, user.Username, deviceToken)
	messageID, err := fcmClient.Send(context.Background(), message)
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		// Optionally, you can log the error to a file or monitoring system
		// logToFile(err)
		// logToMonitoringSystem(err)
		return
	}

	log.Printf("Notification sent successfully: type=%s, messageID=%s", req.CallType, messageID)
	ctx.JSON(http.StatusOK, noticeResponse{
		Code: 0,
		Data: "",
		Msg:  "success",
	})
}
