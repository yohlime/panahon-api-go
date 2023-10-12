package util

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

type Float4 struct {
	pgtype.Float4
}

// MarshalJSON for NullFloat4
func (src Float4) MarshalJSON() ([]byte, error) {
	if src.Valid {
		return json.Marshal(src.Float32)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON for NullFloat4
func (dst *Float4) UnmarshalJSON(b []byte) error {
	var f *float32
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}

	if f == nil {
		dst.Valid = false
	} else {
		dst.Float32 = *f
		dst.Valid = true
	}

	return nil
}
