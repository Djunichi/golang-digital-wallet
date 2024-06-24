package controllers

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"user-service/common/requests"
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

// CreateUser
// @Summary Create a new user
// @Description Create a new user with the provided email
// @Tags users
// @Accept json
// @Produce json
// @Param request body requests.CreateUserRequest true "User email"
// @Success 200 {object} responses.BaseResponse
// @Failure 400 {object} responses.BaseResponse
// @Failure 500 {object} responses.BaseResponse
// @Router /createUser [post]
func (userController *UserController) CreateUser(c *gin.Context) {
	var request requests.CreateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)

	go userController.processCreateUserRequest(request, result)

	response := <-result
	status, ok := response["status"].(int)
	if !ok {
		status = http.StatusOK
	}
	c.JSON(status, response)
}

// GetBalance
// @Summary Get user balance
// @Description Get the balance of a user by email
// @Tags users
// @Produce json
// @Param email path string true "User email"
// @Success 200 {object} responses.GetBalanceResponse
// @Failure 400 {object} responses.BaseResponse
// @Failure 404 {object} responses.BaseResponse
// @Failure 500 {object} responses.BaseResponse
// @Router /balance/{email} [get]
func (userController *UserController) GetBalance(c *gin.Context) {
	email := c.Param("email")

	decodedEmail, err := url.PathUnescape(email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	result := make(chan gin.H)

	go userController.processBalanceRequest(decodedEmail, result)

	response := <-result
	status, ok := response["status"].(int)
	if !ok {
		status = http.StatusOK
	}
	c.JSON(status, response)
}

func (userController *UserController) processBalanceRequest(email string, result chan gin.H) {
	defer close(result)

	tx, err := userController.client.Tx(context.Background())
	if err != nil {
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "Transaction error: " + err.Error(),
		}
		return
	}

	_, err = tx.User.Query().Where(user.EmailEQ(email)).Only(context.Background())
	if err != nil {
		result <- gin.H{
			"status":  http.StatusNotFound,
			"message": "User not found",
		}
		return
	}

	msg, err := userController.natsConn.Request("get-balance", []byte(email), 1000*time.Second)
	if err != nil {
		result <- gin.H{
			"status": http.StatusInternalServerError,
			"error":  "NATS request error: " + err.Error(),
		}
		return
	}

	balance := string(msg.Data)
	bal, err := strconv.ParseFloat(balance, 64)
	if err != nil {
		result <- gin.H{
			"status": http.StatusInternalServerError,
			"error":  "NATS request error: " + balance,
		}
	}

	result <- gin.H{
		"status":  http.StatusOK,
		"balance": bal,
	}
}

func (userController *UserController) processCreateUserRequest(request requests.CreateUserRequest, result chan gin.H) {
	defer close(result)

	tx, err := userController.client.Tx(context.Background())
	if err != nil {
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "Transaction error: " + err.Error(),
		}
		return
	}
	_, err = tx.User.Query().Where(user.EmailEQ(request.Email)).Only(context.Background())
	if err == nil {
		result <- gin.H{
			"status":  http.StatusBadRequest,
			"message": "User already exists",
		}
		return
	}

	u, err := tx.User.Create().
		SetEmail(request.Email).
		SetCreatedAt(time.Now()).
		Save(context.Background())
	if err != nil {
		tx.Rollback()
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "Database error: " + err.Error(),
		}
		return
	}

	userData, err := json.Marshal(u)
	if err != nil {
		tx.Rollback()
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "JSON marshal error: " + err.Error(),
		}
		return
	}

	msg, err := userController.natsConn.Request("user-created", userData, 10*time.Second)
	if err != nil {
		tx.Rollback()
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "NATS request error: " + err.Error(),
		}
		return
	}

	var response gin.H
	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		tx.Rollback()
		result <- gin.H{
			"status":  http.StatusInternalServerError,
			"message": "JSON unmarshal error: " + err.Error(),
		}
		return
	}

	tx.Commit()
	result <- response
}
