// model/book.go
package model

import "time"

type Book struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Category          string  `json:"category"`
	RentalCost        float64 `json:"rental_cost"`
	StockAvailability int64   `json:"stock_availability"`
}

type BookItemStatus string

const (
	ItemAvailable BookItemStatus = "AVAILABLE"
	ItemBooked    BookItemStatus = "BOOKED"
	ItemRented    BookItemStatus = "RENTED"
)

type BookItem struct {
	ID          int64          `json:"id"`
	BookID      int64          `json:"book_id"`
	Status      BookItemStatus `json:"status"`
	BookedUntil *time.Time     `json:"booked_until,omitempty"`
}
