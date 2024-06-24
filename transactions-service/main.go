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

// @title Golang Digital Wallet Transaction Service
// @version 1.0
// @description This is a sample Transaction Service for a digital wallet.

// @host localhost:8081
// @BasePath /
func main() {
	client, err := initializeDatabase()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer client.Close()

	natsConn, err := initializeNATS()
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	messaging.SetupNATS(natsConn, client)

	r := setupRouter(client, natsConn)

	if err := r.Run(":8083"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

// initializeDatabase Initialize DB
func initializeDatabase() (*ent.Client, error) {
	dbURL := os.Getenv("DATABASE_URL")
	dbProvider := os.Getenv("DB_PROVIDER")

	client, err := ent.Open(dbProvider, dbURL)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// initializeNATS Initialize NATS
func initializeNATS() (*nats.Conn, error) {
	natsURL := os.Getenv("NATS_URL")
	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	return natsConn, nil
}

// setupRouter Routing
func setupRouter(client *ent.Client, natsConn *nats.Conn) *gin.Engine {
	r := gin.Default()

	transactionsController := controllers.NewTransactionsController(client, natsConn)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/addMoney", transactionsController.AddMoney)
		v1.POST("/transferMoney", transactionsController.TransferMoney)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
