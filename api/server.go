package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/token"
	"github.com/net-cyber/neka_pay/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// this server will serves HTTP requests for our banking system
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
	cloudinary *util.CloudinaryService
}

// NewServer will create a new HTTP server and set up routing
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	// Initialize Cloudinary service
	cloudinary, err := util.NewCloudinaryService(config.CloudinaryURL)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize cloudinary: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		cloudinary: cloudinary,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}
	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Add metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/tokens/renew_access", server.renewAccessToken)

	// Add phone verification endpoints
	router.POST("/verify/send", server.sendVerificationCode)
	router.POST("/verify/resend", server.resendVerificationCode)
	router.POST("/verify/confirm", server.verifyPhoneNumber)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccount)
	authRoutes.GET("/accounts/verification/:id", server.getAccountForVerification)
	authRoutes.GET("/accounts", server.listAccounts)

	authRoutes.POST("/transfers", server.CreateTransfer)
	authRoutes.POST("/accounts/:id/topup", server.topUpAccount)

	// Add financial institution routes - now all under regular auth routes
	authRoutes.GET("/financial-institutions", server.listFinancialInstitutions)
	authRoutes.GET("/financial-institutions/:id", server.getFinancialInstitution)
	authRoutes.POST("/financial-institutions", server.createFinancialInstitution)
	authRoutes.PUT("/financial-institutions/:id", server.updateFinancialInstitution)

	server.router = router
}

func (server Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}

// Role-based middleware for banker authorization
func authBankerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
		if authPayload.Role != util.BankerRole {
			err := errors.New("only bankers can access this resource")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.Next()
	}
}
