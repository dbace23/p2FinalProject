package userrepo

import (
	"context"

	"instagram/model"
	"instagram/util/database"
)

type Repo interface {
	Create(ctx context.Context, u *model.User) error
	ByEmail(ctx context.Context, email string) (*model.User, error)
}

type repo struct{ db *database.DB }

func New(db *database.DB) Repo { return &repo{db} }

func (r *repo) Create(ctx context.Context, u *model.User) error {
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO users(first_name, last_name, email, username, password_hash)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		u.FirstName, u.LastName, u.Email, u.Username, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *repo) ByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	err := r.db.Pool.QueryRow(ctx, `
        SELECT id, first_name, last_name, email, username, password_hash, created_at
        FROM users
        WHERE lower(email) = lower($1)`,
		email,
	).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
