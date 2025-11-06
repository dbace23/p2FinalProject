package wallet

type CreateTopupReq struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
}
