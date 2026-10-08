// Package auth: password hashing and JWT helpers (standard library only).
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Iterations follows the OWASP recommendation for PBKDF2-HMAC-SHA256. Tests lower it.
var Iterations = 600000

var b64 = base64.RawStdEncoding

func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk, err := pbkdf2.Key(sha256.New, pw, salt, Iterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", Iterations, b64.EncodeToString(salt), b64.EncodeToString(dk)), nil
}

func VerifyPassword(pw, enc string) bool {
	p := strings.Split(enc, "$")
	if len(p) != 4 || p[0] != "pbkdf2-sha256" {
		return false
	}
	it, err := strconv.Atoi(p[1])
	if err != nil || it < 1 {
		return false
	}
	salt, err1 := b64.DecodeString(p[2])
	want, err2 := b64.DecodeString(p[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, it, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

func RandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
