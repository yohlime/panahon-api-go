package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/exp/constraints"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// RandomInt generates a random integer between min and max
func RandomInt[T constraints.Integer](min, max T) T {
	return min + T(rand.Int63n(int64(max-min)+1))
}

// RandomFloat generates a random float between min and max
func RandomFloat[T constraints.Float](min, max T) T {
	return min + T(rand.Float32())*(max-min)
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}
