package util

import (
	"fmt"
	"regexp"
	"time"
)

// ParseDateTime extracts a valid mobile number from s.
func ParseDateTime(s string) (time.Time, bool) {
	match := DateTimeRegExp().FindString(s)

	var dateTimeStr string
	switch matchLen := len(match); matchLen {
	case 10:
		dateTimeStr = fmt.Sprintf("%sT00:00:00+08:00", match)
	case 19:
		dateTimeStr = fmt.Sprintf("%s+08:00", match)
	case 25:
		dateTimeStr = match
	default:
		return time.Time{}, false
	}

	dt, err := time.Parse(time.RFC3339, dateTimeStr)
	if err != nil {
		return time.Time{}, false
	}

	return dt, true
}

func DateTimeRegExp() *regexp.Regexp {
	dateRegExp := "\\d{4}(-\\d{2}){2}"
	timeRegExp := "\\d{2}(:\\d{2}){2}"
	timeOffRegExp := "[\\+-]\\d{2}:\\d{2}"
	dateTimeRegexp := fmt.Sprintf("^(%s)(T%s(%s)?)?$", dateRegExp, timeRegExp, timeOffRegExp)
	return regexp.MustCompile(dateTimeRegexp)
}
