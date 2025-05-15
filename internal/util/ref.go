package util

import (
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func ToRef[T Number](val T) *T {
	return &val
}
