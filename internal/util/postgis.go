package util

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
)

type Point struct {
	*geom.Point
}

// Scan implements the database/sql Scanner interface.
func (p *Point) Scan(src interface{}) error {
	if src == nil {
		p.Point = nil
		return nil
	}

	var b []byte
	var err error

	switch v := src.(type) {
	case string:
		// For pgx, decode the hex-encoded string into bytes
		b, err = hex.DecodeString(v)
		if err != nil {
			return err
		}
	case []byte:
		// For lib/pq, cast it to string and decode the hex-encoded string into bytes
		b, err = hex.DecodeString(string(v))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	got, err := ewkb.Unmarshal(b)
	if err != nil {
		return err
	}
	p1, ok := got.(*geom.Point)
	if !ok {
		return fmt.Errorf("unsupported type: %T", got)
	}
	p.Point = p1
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (p *Point) Value() (driver.Value, error) {
	if p.Point == nil {
		return nil, nil
	}
	sb := &strings.Builder{}
	if err := ewkb.Write(sb, ewkb.NDR, p.Point); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}
