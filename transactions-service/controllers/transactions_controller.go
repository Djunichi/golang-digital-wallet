package controllers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"time"
	"transactions-service/common/requests"
	"transactions-service/ent"
	"transactions-service/ent/transaction"
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

// AddMoney godoc
// @Summary Add money to a user's account
// @Description Add a specified amount of money to a user's account balance
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body requests.AddMoneyRequest true "Add Money Request"
// @Success 200 {object} responses.AddMoneyResponse
// @Failure 400 {object} responses.BaseResponse
// @Failure 500 {object} responses.BaseResponse
// @Router /addMoney [post]
func (ctrl *TransactionsController) AddMoney(c *gin.Context) {
	var req requests.AddMoneyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)
	go ctrl.processAddMoneyRequest(req, result)

	response := <-result
	status, ok := response["status"].(int)
	if !ok {
		status = http.StatusOK
	}
	c.JSON(status, response)
}

// TransferMoney godoc
// @Summary Transfer money between two users
// @Description Transfer a specified amount of money from one user's account to another
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body requests.TransferMoneyRequest true "Transfer Money Request"
// @Success 200 {object} responses.BaseResponse
// @Failure 400 {object} responses.BaseResponse
// @Failure 500 {object} responses.BaseResponse
// @Router /transferMoney [post]
func (ctrl *TransactionsController) TransferMoney(c *gin.Context) {
	var req requests.TransferMoneyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	result := make(chan gin.H)
	go ctrl.processTransferMoneyRequest(req, result)

	response := <-result
	status, ok := response["status"].(int)
	if !ok {
		status = http.StatusOK
	}
	c.JSON(status, response)
}

func (ctrl *TransactionsController) processAddMoneyRequest(req requests.AddMoneyRequest, result chan gin.H) {
	defer close(result)

	ctx := context.Background()
	tx, err := ctrl.client.Tx(ctx)
	if err != nil {
		sendErrorResponse(result, "error creating transaction: "+err.Error())
		return
	}

	if isRequestProcessed(ctx, tx, req.RequestId) {
		sendErrorResponse(result, "request already processed")
		return
	}

	u, err := ctrl.updateUserBalance(ctx, tx, req.UserID, req.Amount)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, err.Error())
		return
	}

	err = ctrl.createTransactionRecord(ctx, tx, req.UserID, req.Amount, req.RequestId)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, "error creating transaction record: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		sendErrorResponse(result, "error committing transaction: "+err.Error())
		return
	}

	result <- gin.H{
		"status":          http.StatusOK,
		"updated_balance": u.Balance,
	}
}

func (ctrl *TransactionsController) processTransferMoneyRequest(req requests.TransferMoneyRequest, result chan gin.H) {
	defer close(result)

	ctx := context.Background()
	tx, err := ctrl.client.Tx(ctx)
	if err != nil {
		sendErrorResponse(result, "error creating transaction: "+err.Error())
		return
	}

	if isRequestProcessed(ctx, tx, req.RequestId) {
		sendErrorResponse(result, "request already processed")
		return
	}

	_, err = ctrl.updateUserBalance(ctx, tx, req.FromUserID, -req.AmountToTransfer)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, "error updating from user balance: "+err.Error())
		return
	}

	_, err = ctrl.updateUserBalance(ctx, tx, req.ToUserID, req.AmountToTransfer)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, "error updating to user balance: "+err.Error())
		return
	}

	err = ctrl.createTransactionRecord(ctx, tx, req.FromUserID, -req.AmountToTransfer, req.RequestId)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, "error creating from user transaction record: "+err.Error())
		return
	}

	err = ctrl.createTransactionRecord(ctx, tx, req.ToUserID, req.AmountToTransfer, req.RequestId)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(result, "error creating to user transaction record: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		sendErrorResponse(result, "error committing transaction: "+err.Error())
		return
	}

	result <- gin.H{
		"status":  http.StatusOK,
		"message": "Money transferred successfully",
	}
}

func (ctrl *TransactionsController) updateUserBalance(ctx context.Context, tx *ent.Tx, userID int, amount float64) (*ent.User, error) {
	u, err := tx.User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		return nil, err
	}

	if amount < 0 && u.Balance+amount < 0 {
		err = errors.New("insufficient funds for transfer")
		return nil, err
	}

	u, err = tx.User.UpdateOneID(userID).AddBalance(amount).Save(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (ctrl *TransactionsController) createTransactionRecord(ctx context.Context, tx *ent.Tx, userID int, amount float64, requestId uuid.UUID) error {
	var t transaction.Type
	if amount > 0 {
		t = "credit"
	} else {
		t = "debit"
	}

	_, err := tx.Transaction.Create().
		SetUserID(userID).
		SetAmount(amount).
		SetCreatedAt(time.Now()).
		SetRequestID(requestId).
		SetType(t).
		Save(ctx)
	return err
}

func sendErrorResponse(result chan gin.H, message string) {
	result <- gin.H{
		"status":  http.StatusInternalServerError,
		"message": message,
	}
}

func isRequestProcessed(ctx context.Context, tx *ent.Tx, requestID uuid.UUID) bool {
	exists, _ := tx.Transaction.Query().Where(transaction.RequestIDEQ(requestID)).Exist(ctx)
	return exists
}
