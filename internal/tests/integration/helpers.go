//go:build integration

package integration

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("integration: read random bytes: %v", err))
	}
	return hex.EncodeToString(buf)
}
