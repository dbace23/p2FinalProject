package book

type CreateBookReq struct {
	Name       string  `json:"name" validate:"required"`
	Category   string  `json:"category" validate:"required"`
	RentalCost float64 `json:"rental_cost" validate:"required,gte=0"`
}

type AddCopiesReq struct {
	Count int `json:"count" validate:"required,gt=0"`
}
