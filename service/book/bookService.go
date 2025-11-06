package booksvc

import (
	"context"
	"errors"

	repo "bookrental/repository/book"
)

type Book = repo.Book

type Repo interface {
	CreateBook(ctx context.Context, name, category string, cost float64) (int64, error)
	AddCopies(ctx context.Context, bookID int64, n int) (int64, error)
	List(ctx context.Context) ([]Book, error)
	Detail(ctx context.Context, id int64) (*Book, error)
}

type Service interface {
	Create(ctx context.Context, name, category string, cost float64) (int64, error)
	AddCopies(ctx context.Context, bookID int64, n int) (int64, error)
	List(ctx context.Context) ([]Book, error)
	Detail(ctx context.Context, id int64) (*Book, error)
}

type service struct{ r Repo }

func New(r Repo) Service { return &service{r: r} }

func (s *service) Create(ctx context.Context, name, category string, cost float64) (int64, error) {
	if name == "" || category == "" || cost < 0 {
		return 0, errors.New("invalid payload")
	}
	return s.r.CreateBook(ctx, name, category, cost)
}
func (s *service) AddCopies(ctx context.Context, bookID int64, n int) (int64, error) {
	return s.r.AddCopies(ctx, bookID, n)
}
func (s *service) List(ctx context.Context) ([]Book, error)            { return s.r.List(ctx) }
func (s *service) Detail(ctx context.Context, id int64) (*Book, error) { return s.r.Detail(ctx, id) }
