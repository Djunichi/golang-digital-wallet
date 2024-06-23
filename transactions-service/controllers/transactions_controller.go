package controllers

import (
	"context"
	"net/http"
	"time"
	"transactions-service/ent"
	"transactions-service/ent/user"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

type TransactionsController struct {
	client *ent.Client
	nc     *nats.Conn
}

func NewTransactionsController(client *ent.Client, natsConn *nats.Conn) *TransactionsController {
	return &TransactionsController{client: client, nc: natsConn}
}

type AddMoneyRequest struct {
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
}

func (transactionController *TransactionsController) AddMoney(c *gin.Context) {
	var req AddMoneyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)
	go func() {
		tx, err := transactionController.client.Tx(context.Background())
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		userTo := tx.User.Query().Where(user.IDEQ(req.UserID))
		if userTo == nil {
			result <- gin.H{
				"status":  "error",
				"message": "From user not found",
			}
			return
		}

		u, err := tx.User.UpdateOneID(req.UserID).AddBalance(req.Amount).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		_, err = tx.Transaction.Create().SetUserID(req.UserID).SetAmount(req.Amount).SetCreatedAt(time.Now()).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		tx.Commit()
		result <- gin.H{
			"updated_balance": u.Balance,
		}
	}()

	c.JSON(http.StatusOK, <-result)
}

type TransferMoneyRequest struct {
	FromUserID       int     `json:"from_user_id"`
	ToUserID         int     `json:"to_user_id"`
	AmountToTransfer float64 `json:"amount_to_transfer"`
}

func (transactionController *TransactionsController) TransferMoney(c *gin.Context) {
	var req TransferMoneyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)
	go func() {
		tx, err := transactionController.client.Tx(context.Background())
		if err != nil {
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		fromUser := tx.User.Query().Where(user.IDEQ(req.FromUserID))
		if fromUser == nil {
			result <- gin.H{
				"status":  "error",
				"message": "From user not found",
			}
			return
		}

		toUser := tx.User.Query().Where(user.IDEQ(req.ToUserID))
		if toUser == nil {
			result <- gin.H{
				"status":  "error",
				"message": "To user not found",
			}
			return
		}

		_, err = tx.User.UpdateOneID(req.FromUserID).AddBalance(-req.AmountToTransfer).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		_, err = tx.User.UpdateOneID(req.ToUserID).AddBalance(req.AmountToTransfer).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		_, err = tx.Transaction.Create().SetUserID(req.FromUserID).SetAmount(-req.AmountToTransfer).SetCreatedAt(time.Now()).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		_, err = tx.Transaction.Create().SetUserID(req.ToUserID).SetAmount(req.AmountToTransfer).SetCreatedAt(time.Now()).Save(context.Background())
		if err != nil {
			tx.Rollback()
			result <- gin.H{
				"status":  "error",
				"message": err.Error(),
			}
			return
		}

		tx.Commit()
		result <- gin.H{
			"status":  "success",
			"message": "Money transferred successfully",
		}
	}()

	c.JSON(http.StatusOK, <-result)
}
