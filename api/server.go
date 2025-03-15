package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
)

// this server will serves HTTP requests for our banking system
type Server struct {
	store *db.Store
	router *gin.Engine
}

// NewServer will create a new HTTP server and set up routing
func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// add routers to router
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccount)
	server.router = router
	return server
}

func (server Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{
		"error" : err.Error(),
	}
}