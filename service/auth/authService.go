package auth

import (
	"context"
	"errors"
	"strings"

	"bookrental/model"
	authrepo "bookrental/repository/auth"
	"bookrental/util/hash"
	"bookrental/util/jwt"
)

type ErrCode string

const (
	ErrEmailTaken    ErrCode = "EMAIL_TAKEN"
	ErrUsernameTaken ErrCode = "USERNAME_TAKEN"
	ErrInvalidCreds  ErrCode = "INVALID_CREDS"
	ErrBadInput      ErrCode = "BAD_INPUT"
)

type codedError struct {
	code ErrCode
	msg  string
}

func (e codedError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return string(e.code)
}
func (e codedError) Code() ErrCode { return e.code }

func wrap(code ErrCode, msg string) error { return codedError{code: code, msg: msg} }

func Code(err error) ErrCode {
	var ce interface{ Code() ErrCode }
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return ""
}

type Service interface {
	Register(ctx context.Context, req model.RegisterReq) (*model.User, string, error)
	Login(ctx context.Context, req model.LoginReq) (*model.User, string, error)
}

type service struct {
	repo      authrepo.Repo
	jwtSecret string
	ttlHours  int
}

func New(r authrepo.Repo, jwtSecret string) Service {
	return &service{repo: r, jwtSecret: jwtSecret, ttlHours: 24}
}

func (s *service) Register(ctx context.Context, req model.RegisterReq) (*model.User, string, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	if req.Email == "" || req.Username == "" || len(req.Password) < 6 {
		return nil, "", wrap(ErrBadInput, "invalid input")
	}

	if existing, _ := s.repo.ByEmail(ctx, req.Email); existing != nil && existing.ID > 0 {
		return nil, "", wrap(ErrEmailTaken, "email already registered")
	}

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
		Role:         "user",
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, "", err
	}

	tok, err := jwt.Issue(s.jwtSecret, int64(u.ID), u.Role, s.ttlHours)
	if err != nil {
		return nil, "", err
	}
	return u, tok, nil
}

func (s *service) Login(ctx context.Context, req model.LoginReq) (*model.User, string, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return nil, "", wrap(ErrBadInput, "invalid input")
	}

	u, err := s.repo.ByEmail(ctx, req.Email)
	if err != nil || u == nil || u.ID == 0 {
		return nil, "", wrap(ErrInvalidCreds, "invalid email or password")
	}
	if !hash.Check(u.PasswordHash, req.Password) {
		return nil, "", wrap(ErrInvalidCreds, "invalid email or password")
	}

	tok, err := jwt.Issue(s.jwtSecret, int64(u.ID), u.Role, s.ttlHours)
	if err != nil {
		return nil, "", err
	}
	return u, tok, nil
}
