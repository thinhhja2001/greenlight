package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"greentlight.thinhhja.net/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

var jwtSecretKey = []byte("nrqEUaC0g3")

type Token struct {
	Plaintext string `json:"token"`
	UserID    int64  `json:"-"`
	Scope     string `json:"scope"`
}

type Payload struct {
	UserID int64     `json:"user_id"`
	Expiry time.Time `json:"expiry"`
}

func (p Payload) Valid() error {
	if p.Expiry.Before(time.Now()) {
		return fmt.Errorf("token expired")
	}
	return nil
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	claims := &Payload{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return nil, err
	}
	return &Token{Plaintext: t,
		UserID: userID,
		Scope:  scope,
	}, nil
}

func ValidateJWTToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t_ *jwt.Token) (interface{}, error) {
		if _, ok := t_.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t_.Header["alg"])
		}
		return []byte(jwtSecretKey), nil
	})
}

func ValidateTokenPlainText(v *validator.Validator, tokenPlainText string) {
	v.Check(tokenPlainText != "", "token", "must be provided")
}

type TokenModel struct {
	DB *sql.DB
}

func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	fmt.Sprintln(token)
	err = m.Insert(token)
	return token, err
}

func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, scope)
		VALUES ($1, $2, $3)
	`
	args := []interface{}{token.Plaintext, token.UserID, token.Scope}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userID)
	return err
}
