package util

import "math"

func FahrenheitToCelsius(f float32) float32 {
	v := (f - 32.0) * (5.0 / 9.0)
	return float32(math.Round(float64(v)))
}

func InHgToMbar(f float32) float32 {
	v := f * 33.86
	return float32(math.Round(float64(v)))
}
