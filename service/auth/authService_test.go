// service/auth/auth_service_test.go
package auth

import (
	"context"
	"errors"
	"testing"

	"bookrental/model"
	authrepo "bookrental/repository/auth"
	"bookrental/util/hash"

	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	byEmailFn func(ctx context.Context, email string) (*model.User, error)
	createFn  func(ctx context.Context, u *model.User) error
}

var _ authrepo.Repo = (*mockRepo)(nil)

func (m *mockRepo) ByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.byEmailFn == nil {
		return nil, nil
	}
	return m.byEmailFn(ctx, email)
}

func (m *mockRepo) Create(ctx context.Context, u *model.User) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(ctx, u)
}

func mustHash(t *testing.T, plain string) string {
	t.Helper()

	h, err := hashPasswordForTest(plain)
	require.NoError(t, err)
	return h
}

func hashPasswordForTest(pw string) (string, error) {
	return hash.HashPassword(pw)
}

// --- tests ---

func TestRegister_Success(t *testing.T) {
	ctx := context.Background()
	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, u *model.User) error {

			u.ID = 42
			return nil
		},
	}
	svc := New(m, "test-secret")

	req := model.RegisterReq{
		FirstName: "Halim",
		LastName:  "Iskandar",
		Email:     "USER@Example.COM",
		Username:  "halim",
		Password:  "supersecret",
	}

	u, tok, err := svc.Register(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotEmpty(t, tok)
	require.Equal(t, int64(42), u.ID)
	require.Equal(t, "user@example.com", u.Email)
	require.Equal(t, "halim", u.Username)
	require.Equal(t, "user", u.Role)
	require.NotEmpty(t, u.PasswordHash)
}

func TestRegister_BadInput(t *testing.T) {
	ctx := context.Background()
	svc := New(&mockRepo{}, "test-secret")

	_, _, err := svc.Register(ctx, model.RegisterReq{
		Email:    " ",
		Username: "u",
		Password: "123",
	})
	require.Error(t, err)
	require.Equal(t, ErrBadInput, Code(err))
}

func TestRegister_EmailTaken(t *testing.T) {
	ctx := context.Background()
	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return &model.User{ID: 9, Email: email}, nil
		},
	}
	svc := New(m, "test-secret")

	_, _, err := svc.Register(ctx, model.RegisterReq{
		Email:    "taken@example.com",
		Username: "halim",
		Password: "123456",
	})
	require.Error(t, err)
	require.Equal(t, ErrEmailTaken, Code(err))
}

func TestRegister_CreateError(t *testing.T) {
	ctx := context.Background()
	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, u *model.User) error {
			return errors.New("db down")
		},
	}
	svc := New(m, "test-secret")

	_, _, err := svc.Register(ctx, model.RegisterReq{
		Email:    "ok@example.com",
		Username: "ok",
		Password: "123456",
	})
	require.Error(t, err)

	require.Equal(t, ErrCode(""), Code(err))
}

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	pw := "supersecret"
	hashed := mustHash(t, pw)

	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return &model.User{
				ID:           7,
				Email:        "user@example.com",
				Username:     "halim",
				PasswordHash: hashed,
				Role:         "user",
			}, nil
		},
	}
	svc := New(m, "test-secret")

	u, tok, err := svc.Login(ctx, model.LoginReq{
		Email:    "User@Example.com",
		Password: pw,
	})
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotEmpty(t, tok)
	require.Equal(t, int64(7), u.ID)
}

func TestLogin_BadInput(t *testing.T) {
	ctx := context.Background()
	svc := New(&mockRepo{}, "test-secret")

	_, _, err := svc.Login(ctx, model.LoginReq{
		Email:    " ",
		Password: "",
	})
	require.Error(t, err)
	require.Equal(t, ErrBadInput, Code(err))
}

func TestLogin_UserNotFound(t *testing.T) {
	ctx := context.Background()
	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return nil, nil
		},
	}
	svc := New(m, "test-secret")

	_, _, err := svc.Login(ctx, model.LoginReq{
		Email:    "missing@example.com",
		Password: "whatever",
	})
	require.Error(t, err)
	require.Equal(t, ErrInvalidCreds, Code(err))
}

func TestLogin_WrongPassword(t *testing.T) {
	ctx := context.Background()

	hashed := mustHash(t, "correct-password")

	m := &mockRepo{
		byEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return &model.User{
				ID:           101,
				Email:        "user@example.com",
				Username:     "halim",
				PasswordHash: hashed,
				Role:         "user",
			}, nil
		},
	}
	svc := New(m, "test-secret")

	_, _, err := svc.Login(ctx, model.LoginReq{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	require.Error(t, err)
	require.Equal(t, ErrInvalidCreds, Code(err))
}

func TestCodeExtractor(t *testing.T) {
	require.Equal(t, ErrEmailTaken, Code(wrap(ErrEmailTaken, "x")))
	require.Equal(t, ErrCode(""), Code(errors.New("plain")))
}
