package authsvc

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"

	"instagram/model"
	userrepo "instagram/repository/user"
	"instagram/util/hash"
	jwtutil "instagram/util/jwt"

	"github.com/jackc/pgconn"
)

var (
	ErrEmailTaken    = errors.New("email already registered")
	ErrBadInput      = errors.New("bad input")
	ErrInvalidCreds  = errors.New("invalid credentials")
	ErrUsernameTaken = errors.New("username already taken")
)

type Service interface {
	Register(ctx context.Context, req model.RegisterReq, secret string) (*model.User, string, error)
	Login(ctx context.Context, req model.LoginReq, secret string) (*model.User, string, error)
}

type service struct{ ur userrepo.Repo }

func New(ur userrepo.Repo) Service { return &service{ur} }

func (s *service) Register(ctx context.Context, req model.RegisterReq, secret string) (*model.User, string, error) {
	hashed, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, "", err
	}

	u := &model.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hashed,
	}

	if err := s.ur.Create(ctx, u); err != nil {
		if derr := mapDuplicateErr(err); derr != nil {
			return nil, "", derr
		}
		return nil, "", err
	}

	token, err := jwtutil.Issue(secret, u.ID, "user", 24)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func mapDuplicateErr(err error) error {

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		cn := strings.ToLower(pgErr.ConstraintName)
		msg := strings.ToLower(pgErr.Message)

		if strings.Contains(cn, "users_email") || strings.Contains(msg, "email") {
			return ErrEmailTaken
		}
		if strings.Contains(cn, "users_username") || strings.Contains(msg, "username") {
			return ErrUsernameTaken
		}
		return ErrBadInput
	}

	return nil
}

func (s *service) Login(ctx context.Context, req model.LoginReq, secret string) (*model.User, string, error) {
	u, err := s.ur.ByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", ErrInvalidCreds
	}
	if !hash.Check(u.PasswordHash, req.Password) {
		return nil, "", ErrInvalidCreds
	}
	token, err := jwtutil.Issue(secret, u.ID, "user", 24)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}
