package api

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validMobileNumber validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if mobileNumber, ok := fieldLevel.Field().Interface().(string); ok {
		return regexp.MustCompile("^+[0-9]{12}$").MatchString(mobileNumber)
	}
	return false
}
