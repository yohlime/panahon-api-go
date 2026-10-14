package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Different types of error returned by the VerifyToken function
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

type User struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// Payload contains the payload data of the token
type Payload struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id,omitempty"`
	User      User      `json:"user"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewPayload creates a new token payload with a specific username and duration.
// sessionID binds the token to a server-side session (uuid.Nil for legacy
// tokens that predate session binding).
func NewPayload(user User, sessionID uuid.UUID, duration time.Duration) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &Payload{
		ID:        tokenID,
		SessionID: sessionID,
		User:      user,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}

	return payload, nil
}

// Valid checks if the token is valid or not
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiresAt) {
		return ErrExpiredToken
	}
	return nil
}
