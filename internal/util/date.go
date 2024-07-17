package util

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// Date represents a date-only value
type Date struct {
	time.Time
}

// Layout used for parsing and formatting dates
const layout = "2006-01-02"

// UnmarshalJSON implements the json.Unmarshaler interface
func (d *Date) UnmarshalJSON(data []byte) error {
	// Unquote the JSON string to get the date string
	str, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	// Parse the date string to time.Time
	parsedTime, err := time.Parse(layout, str)
	if err != nil {
		return err
	}
	*d = Date{Time: parsedTime}
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (d Date) MarshalJSON() ([]byte, error) {
	// Format the date to the desired layout
	formattedDate := d.Format(layout)
	// Quote the formatted date string and return as JSON
	return json.Marshal(formattedDate)
}

// String implements the Stringer interface for Date
func (d Date) String() string {
	return d.Format(layout)
}

// Fake implements the gofakeit.Faker interface
func (c *Date) Fake(f *gofakeit.Faker) (any, error) {
	t := f.Date()
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())}, nil
}
