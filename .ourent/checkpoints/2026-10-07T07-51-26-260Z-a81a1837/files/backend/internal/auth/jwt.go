package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Sub  int64  `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
}

var ErrToken = errors.New("invalid token")
var enc = base64.RawURLEncoding

const header = `{"alg":"HS256","typ":"JWT"}`

func mac(secret, data string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(data))
	return enc.EncodeToString(m.Sum(nil))
}

func Sign(secret string, uid int64, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	body, err := json.Marshal(Claims{Sub: uid, Role: role, Iat: now.Unix(), Exp: now.Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	data := enc.EncodeToString([]byte(header)) + "." + enc.EncodeToString(body)
	return data + "." + mac(secret, data), nil
}

func Parse(secret, tok string) (*Claims, error) {
	p := strings.Split(tok, ".")
	if len(p) != 3 || p[0] != enc.EncodeToString([]byte(header)) { // pins alg=HS256, rejects "none"
		return nil, ErrToken
	}
	if !hmac.Equal([]byte(p[2]), []byte(mac(secret, p[0]+"."+p[1]))) {
		return nil, ErrToken
	}
	raw, err := enc.DecodeString(p[1])
	if err != nil {
		return nil, ErrToken
	}
	var c Claims
	if json.Unmarshal(raw, &c) != nil || time.Now().Unix() > c.Exp {
		return nil, ErrToken
	}
	return &c, nil
}
