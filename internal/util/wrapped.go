package util

import (
	"fmt"
	"math"
	"strconv"

	"golang.org/x/exp/constraints"
)

type WrappedInt[T constraints.Integer] struct {
	Value T
	Valid bool
}

func NewWrappedInt[T constraints.Integer](s string) *WrappedInt[T] {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &WrappedInt[T]{Valid: false}
	}
	return &WrappedInt[T]{
		Value: T(val),
		Valid: true,
	}
}

func (f WrappedInt[T]) GetRef() *T {
	if !f.Valid {
		return nil
	}
	return &f.Value
}

type WrappedFloat[T constraints.Float] struct {
	Value    T
	Valid    bool
	Decimals int
}

func NewWrappedFloat[T constraints.Float](s string) *WrappedFloat[T] {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return &WrappedFloat[T]{Valid: false}
	}
	return &WrappedFloat[T]{
		Value: T(val),
		Valid: true,
	}
}

func (f WrappedFloat[T]) Round(decimals int) WrappedFloat[T] {
	if !f.Valid {
		return WrappedFloat[T]{Valid: false}
	}
	val := float64(f.Value)
	scale := math.Pow(10, float64(decimals))
	rounded := math.Round(val*scale) / scale
	return WrappedFloat[T]{
		Value:    T(rounded),
		Valid:    true,
		Decimals: decimals,
	}
}

func (f WrappedFloat[T]) Convert(cf T) WrappedFloat[T] {
	if !f.Valid {
		return WrappedFloat[T]{Valid: false}
	}
	return WrappedFloat[T]{
		Value: T(f.Value * cf),
		Valid: true,
	}
}

func (f WrappedFloat[T]) Validate(fn func(T) bool) WrappedFloat[T] {
	if !f.Valid || !fn(f.Value) {
		return WrappedFloat[T]{Valid: false}
	}
	return f
}

func (f WrappedFloat[T]) GetRef() *T {
	if !f.Valid {
		return nil
	}
	return &f.Value
}

func (f WrappedFloat[T]) String() string {
	if !f.Valid {
		return ""
	}
	format := "%.2f"
	if f.Decimals > 0 {
		format = fmt.Sprintf("%%.%df", f.Decimals)
	}
	return fmt.Sprintf(format, f.Value)
}
