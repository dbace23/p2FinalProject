package rental

type BookWithDepositReq struct {
	BookID      int64 `json:"book_id" validate:"required,gt=0"`
	HoldMinutes *int  `json:"hold_minutes,omitempty" validate:"omitempty,min=0,max=1440"`
}
