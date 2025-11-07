// service/book/book_service_test.go
package booksvc_test

import (
	"context"
	"errors"
	"testing"

	booksvc "bookrental/service/book"
)

type repoMock struct {
	createFn    func(ctx context.Context, name, category string, cost float64) (int64, error)
	addCopiesFn func(ctx context.Context, bookID int64, n int) (int64, error)
	listFn      func(ctx context.Context) ([]booksvc.Book, error)
	detailFn    func(ctx context.Context, id int64) (*booksvc.Book, error)
}

func (m *repoMock) CreateBook(ctx context.Context, name, category string, cost float64) (int64, error) {
	return m.createFn(ctx, name, category, cost)
}
func (m *repoMock) AddCopies(ctx context.Context, bookID int64, n int) (int64, error) {
	return m.addCopiesFn(ctx, bookID, n)
}
func (m *repoMock) List(ctx context.Context) ([]booksvc.Book, error) { return m.listFn(ctx) }
func (m *repoMock) Detail(ctx context.Context, id int64) (*booksvc.Book, error) {
	return m.detailFn(ctx, id)
}

func TestCreate_Validation(t *testing.T) {
	s := booksvc.New(&repoMock{})
	if _, err := s.Create(context.Background(), "", "cat", 10); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := s.Create(context.Background(), "name", "", 10); err == nil {
		t.Fatal("expected error for empty category")
	}
	if _, err := s.Create(context.Background(), "name", "cat", -1); err == nil {
		t.Fatal("expected error for negative cost")
	}
}

func TestCreate_Success(t *testing.T) {
	m := &repoMock{
		createFn: func(ctx context.Context, name, category string, cost float64) (int64, error) {
			if name != "Clean Code" || category != "Prog" || cost != 18000 {
				return 0, errors.New("bad args")
			}
			return 42, nil
		},
	}
	s := booksvc.New(m)
	id, err := s.Create(context.Background(), "Clean Code", "Prog", 18000)
	if err != nil || id != 42 {
		t.Fatalf("got id=%v err=%v; want 42 nil", id, err)
	}
}

func TestPassThroughs(t *testing.T) {
	m := &repoMock{
		addCopiesFn: func(ctx context.Context, bookID int64, n int) (int64, error) { return 3, nil },
		listFn:      func(ctx context.Context) ([]booksvc.Book, error) { return nil, nil },
		detailFn:    func(ctx context.Context, id int64) (*booksvc.Book, error) { return &booksvc.Book{}, nil },
	}
	s := booksvc.New(m)

	if n, err := s.AddCopies(context.Background(), 7, 3); err != nil || n != 3 {
		t.Fatalf("AddCopies got %v %v; want 3 nil", n, err)
	}
	if _, err := s.List(context.Background()); err != nil {
		t.Fatalf("List error: %v", err)
	}
	if _, err := s.Detail(context.Background(), 99); err != nil {
		t.Fatalf("Detail error: %v", err)
	}
}
