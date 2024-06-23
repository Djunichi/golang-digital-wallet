package controllers

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"net/http"
	"time"
	"user-service/ent"
	"user-service/ent/user"
)

type UserController struct {
	client   *ent.Client
	natsConn *nats.Conn
}

func NewUserController(client *ent.Client, natsConn *nats.Conn) *UserController {
	return &UserController{client: client, natsConn: natsConn}
}

type CreateUserRequest struct {
	Email string `json:"email"`
}

type CreateUserResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (userController *UserController) CreateUser(c *gin.Context) {

	var request CreateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)
	go func() {
		u, err := userController.client.User.Create().SetEmail(request.Email).SetCreatedAt(time.Now()).Save(context.Background())
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		userData, err := json.Marshal(u)
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		msg, err := userController.natsConn.Request("user-created", userData, 10*time.Second)
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		var response gin.H
		err = json.Unmarshal(msg.Data, &response)
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		result <- response
	}()

	c.JSON(http.StatusOK, <-result)
}

func (userController *UserController) GetBalance(c *gin.Context) {
	email := c.Param("email")

	result := make(chan gin.H)
	go func() {

		tx, err := userController.client.Tx(context.Background())
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		usr := tx.User.Query().Where(user.EmailEQ(email))
		if usr == nil {
			result <- gin.H{
				"status":  "error",
				"message": "From user not found",
			}
			return
		}

		msg, err := userController.natsConn.Request("get-balance", []byte(email), 10*time.Second)
		if err != nil {
			result <- gin.H{"error": err.Error()}
			return
		}

		var balance int
		err = json.Unmarshal(msg.Data, &balance)
		if err != nil {
			result <- gin.H{"error": err.Error()}
			return
		}

		result <- gin.H{"balance": balance}
	}()

	c.JSON(http.StatusOK, <-result)
}
