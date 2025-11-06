package rental

type CreateRentalReq struct {
	BookID int64 `json:"book_id" validate:"required,gt=0"`
}
