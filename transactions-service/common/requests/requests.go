package requests

import "github.com/google/uuid"

type AddMoneyRequest struct {
	UserID    int       `json:"user_id"`
	Amount    float64   `json:"amount"`
	RequestId uuid.UUID `json:"request_id"`
}

type TransferMoneyRequest struct {
	FromUserID       int       `json:"from_user_id"`
	ToUserID         int       `json:"to_user_id"`
	AmountToTransfer float64   `json:"amount_to_transfer"`
	RequestId        uuid.UUID `json:"request_id"`
}
