package responses

type BaseResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GetBalanceResponse struct {
	Status  string  `json:"status"`
	Balance float64 `json:"balance"`
}
