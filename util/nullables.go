package util

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgtype"
)

type NullFloat4 struct {
	pgtype.Float4
}

// MarshalJSON for NullFloat4
func (src NullFloat4) MarshalJSON() ([]byte, error) {
	if src.Valid {
		return json.Marshal(src.Float32)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON for NullFloat4
func (dst *NullFloat4) UnmarshalJSON(b []byte) error {
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

// String for NullFloat4
func (f NullFloat4) String() string {
	if f.Valid {
		return fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100)
	}
	return ""
}

type NullString struct {
	pgtype.Text
}

// String for NullString
func (s NullString) String() string {
	if s.Valid {
		return s.Text.String
	}
	return ""
}

type NullInt4 struct {
	pgtype.Int4
}

// String for NullInt4
func (i NullInt4) String() string {
	if i.Valid {
		return fmt.Sprintf("%d", i.Int32)
	}
	return ""
}
