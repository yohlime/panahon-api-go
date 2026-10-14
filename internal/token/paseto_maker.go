package token

import (
	"encoding/json"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

// PasetoMaker is a PASETO Token maker
type PasetoMaker struct {
	symmetricKey paseto.V4SymmetricKey
}

// NewPasetoMaker creates a new PasetoMaker
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != 32 {
		return nil, fmt.Errorf("invalid key size: must be exactly 32 characters")
	}

	key, err := paseto.V4SymmetricKeyFromBytes([]byte(symmetricKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create symmetric key: %w", err)
	}

	maker := &PasetoMaker{
		symmetricKey: key,
	}
	return maker, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *PasetoMaker) CreateToken(user User, sessionID uuid.UUID, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(user, sessionID, duration)
	if err != nil {
		return "", payload, err
	}

	claimsJSON, err := json.Marshal(payload)
	if err != nil {
		return "", payload, err
	}

	tok, err := paseto.NewTokenFromClaimsJSON(claimsJSON, nil)
	if err != nil {
		return "", payload, err
	}
	tok.SetIssuedAt(payload.IssuedAt)
	tok.SetExpiration(payload.ExpiresAt)

	token := tok.V4Encrypt(maker.symmetricKey, nil)

	return token, payload, err
}

// VerifyToken checks if the token is valid or not
func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	tok, err := parser.ParseV4Local(maker.symmetricKey, token, nil)
	if err != nil {
		return nil, err
	}

	claimsJSON := tok.ClaimsJSON()

	var payload Payload
	if err := json.Unmarshal(claimsJSON, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}
