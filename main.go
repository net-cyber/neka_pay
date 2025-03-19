package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/net-cyber/neka_pay/api"
	db "github.com/net-cyber/neka_pay/db/sqlc"
	"github.com/net-cyber/neka_pay/util"
)


func main() {
	config, err := util.LoadConfig(".")

	if err != nil {
		log.Fatal("cannot load config", err)
	}


	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server, err := api.NewServer(config,store)

	if err != nil {
		log.Fatal("can not create a server:", err)
	}

	err = server.Start(config.ServerAddress)

	if err != nil {
		log.Fatal("can not start server:", err)
	}
}