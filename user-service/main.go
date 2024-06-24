package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"user-service/controllers"
	"user-service/ent"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "user-service/docs"
)

// @title User Service API
// @version 1.0
// @description This is a sample server for a user service.

// @host localhost:8080
// @BasePath /api/v1
func main() {
	client, natsConn, err := initializeResources()
	if err != nil {
		log.Fatalf("failed to initialize resources: %v", err)
	}
	defer client.Close()
	defer natsConn.Close()

	r := setupRouter(client, natsConn)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

// initializeResources Initialize DB and NATS connections
func initializeResources() (*ent.Client, *nats.Conn, error) {
	natsURL := os.Getenv("NATS_URL")
	dbURL := os.Getenv("DATABASE_URL")
	dbProvider := os.Getenv("DB_PROVIDER")

	client, err := ent.Open(dbProvider, dbURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed opening connection to database: %w", err)
	}

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("failed creating schema resources: %w", err)
	}

	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("failed connecting to NATS: %w", err)
	}

	return client, natsConn, nil
}

// setupRouter Routing
func setupRouter(client *ent.Client, natsConn *nats.Conn) *gin.Engine {
	r := gin.Default()

	userController := controllers.NewUserController(client, natsConn)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/createUser", userController.CreateUser)
		v1.GET("/balance/:email", userController.GetBalance)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
