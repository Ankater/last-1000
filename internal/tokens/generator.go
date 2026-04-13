package tokens

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
