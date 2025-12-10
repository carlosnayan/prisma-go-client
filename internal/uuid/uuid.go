package uuid

import (
	"math/rand"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateUUID generates a UUID v4 without external dependencies
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
// where x is any hexadecimal digit and y is one of 8, 9, a, or b
func GenerateUUID() string {
	uuid := make([]byte, 36)
	template := "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"

	for i, c := range template {
		switch c {
		case 'x':
			uuid[i] = "0123456789abcdef"[rng.Intn(16)]
		case 'y':
			uuid[i] = "89ab"[rng.Intn(4)]
		case '-':
			uuid[i] = '-'
		default:
			uuid[i] = byte(c)
		}
	}

	return string(uuid)
}
