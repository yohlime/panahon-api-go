package handlers

import (
	"regexp"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/go-playground/validator/v10"
)

var validMobileNumber validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if mobileNumber, ok := fieldLevel.Field().Interface().(string); ok {
		return regexp.MustCompile("^+[0-9]{12}$").MatchString(mobileNumber)
	}
	return false
}

var validFullName validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if name, ok := fieldLevel.Field().Interface().(string); ok {
		return regexp.MustCompile(`^[A-Z][a-zA-Z'’-]+(?: [A-Z][a-zA-Z'’-]+)*$`).MatchString(name)
	}
	return false
}

var validSentence validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if s, ok := fieldLevel.Field().Interface().(string); ok {
		return regexp.MustCompile(`^[A-Z][a-zA-Z0-9 ,;:'"()!?-]*[.!?]$`).MatchString(s)
	}
	return false
}

var validDateTimeStr validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if datetime, ok := fieldLevel.Field().Interface().(string); ok {
		return util.DateTimeRegExp().MatchString(datetime)
	}
	return false
}
