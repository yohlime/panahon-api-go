package util

import (
	"encoding/json"
	"time"

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

func ToPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: len(s) > 0}
}

func ToInt4(i *int32) pgtype.Int4 {
	if i != nil {
		return pgtype.Int4{Int32: *i, Valid: true}
	}
	return pgtype.Int4{}
}

func ToFloat4(f *float32) pgtype.Float4 {
	if f != nil {
		return pgtype.Float4{Float32: *f, Valid: true}
	}
	return pgtype.Float4{}
}

func ToPgDate(s string) pgtype.Date {
	if len(s) == 10 {
		_dt, err := time.Parse("2006-01-02", s)
		return pgtype.Date{Time: _dt, Valid: err == nil}
	}

	return pgtype.Date{}
}
