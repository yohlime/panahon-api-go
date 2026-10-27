package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var errInvalidCursor = errors.New("invalid cursor")

type observationCursor struct {
	Timestamp time.Time `json:"ts"`
	ID        int64     `json:"id"`
}

func encodeObservationCursor(timestamp time.Time, id int64) string {
	raw, _ := json.Marshal(observationCursor{Timestamp: timestamp, ID: id})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeObservationCursor(value string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, 0, errInvalidCursor
	}

	var cursor observationCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return time.Time{}, 0, errInvalidCursor
	}
	if cursor.Timestamp.IsZero() || cursor.ID <= 0 {
		return time.Time{}, 0, errInvalidCursor
	}

	return cursor.Timestamp, cursor.ID, nil
}
