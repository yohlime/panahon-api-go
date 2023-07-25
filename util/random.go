package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomNullInt4 generates a random NullInt4 between min and max
func RandomNullInt4(min, max int64) NullInt4 {
	i := RandomInt(min, max)
	return NullInt4{
		Int4: pgtype.Int4{
			Int32: int32(i),
			Valid: true,
		},
	}
}

// RandomFloat generates a random float between min and max
func RandomFloat(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}

// RandomNullFloat4 generates a random NullFloat4 between min and max
func RandomNullFloat4(min, max float32) NullFloat4 {
	f := RandomFloat(min, max)
	return NullFloat4{
		Float4: pgtype.Float4{
			Float32: f,
			Valid:   true,
		},
	}
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
