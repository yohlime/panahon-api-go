package util

import (
	"fmt"
	"regexp"
)

// ParseMobileNumber extracts a valid mobile number from s.
func ParseMobileNumber(s string) (string, bool) {
	reg, err := regexp.Compile(`((\+?63)|0)?([1-9]\d{9})`)
	if err != nil {
		return "", false
	}

	matches := reg.FindStringSubmatch(s)
	numMatch := len(matches)
	if numMatch == 0 {
		return "", false
	}

	return fmt.Sprintf("63%s", matches[numMatch-1]), true
}

func RandomMobileNumber() string {
	return fmt.Sprintf("63%d", RandomInt(9000000000, 9999999999))
}
