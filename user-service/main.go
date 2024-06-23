package main

import (
	"context"
	"log"
	"os"
	"user-service/controllers"
	"user-service/ent"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var client *ent.Client
var natsConn *nats.Conn

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

	r := gin.Default()

	userController := controllers.NewUserController(client, natsConn)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/createUser", userController.CreateUser)
		v1.GET("/balance", userController.GetBalance)

	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
