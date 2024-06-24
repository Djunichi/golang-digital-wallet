package main

import (
	"context"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"os"
	"transactions-service/controllers"
	"transactions-service/ent"
	"transactions-service/messaging"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

var client *ent.Client
var natsConn *nats.Conn

// @title Golang Digital Wallet Transaction Service
// @version 1.0
// @description This is a sample Transaction Service for a digital wallet.

// @host localhost:8081
// @BasePath /
func main() {
	var err error

	natsURL := os.Getenv("NATS_URL")
	dbUrl := os.Getenv("DATABASE_URL")
	dbProvider := os.Getenv("DB_PROVIDER")

	client, err = ent.Open(dbProvider, dbUrl)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}

	defer client.Close()

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	natsConn, err = nats.Connect(natsURL)
	if err != nil {
		log.Fatalln(err)
	}
	defer natsConn.Close()

	messaging.SetupNATS(natsConn, client)

	r := gin.Default()

	transactionsController := controllers.NewTransactionsController(client, natsConn)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/addMoney", transactionsController.AddMoney)
		v1.POST("/transferMoney", transactionsController.TransferMoney)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if err := r.Run(":8083"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
