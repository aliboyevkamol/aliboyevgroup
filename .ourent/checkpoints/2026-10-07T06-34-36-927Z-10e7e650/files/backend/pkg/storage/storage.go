// Package storage abstracts file storage. The local driver is implemented;
// S3 / Cloudflare R2 / DigitalOcean Spaces adapters only need to implement Storage.
package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, key string, r io.Reader) error
	Delete(ctx context.Context, key string) error
	// GetURL returns a public URL for "public/" keys and a time-limited signed URL otherwise.
	GetURL(key string, ttl time.Duration) (string, error)
}

type Local struct {
	Dir, BaseURL string
	secret       []byte
}

func NewLocal(dir, baseURL, secret string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Local{Dir: dir, BaseURL: strings.TrimRight(baseURL, "/"), secret: []byte(secret)}, nil
}

func (l *Local) path(key string) (string, error) {
	c := filepath.Clean("/" + key)
	if strings.Contains(key, "..") || c == "/" {
		return "", fmt.Errorf("invalid key")
	}
	return filepath.Join(l.Dir, c), nil
}

func (l *Local) Upload(_ context.Context, key string, r io.Reader) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *Local) sign(key string, exp int64) string {
	m := hmac.New(sha256.New, l.secret)
	m.Write([]byte(key + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))
}

func (l *Local) GetURL(key string, ttl time.Duration) (string, error) {
	if strings.HasPrefix(key, "public/") {
		return l.BaseURL + "/files/" + key, nil
	}
	exp := time.Now().Add(ttl).Unix()
	return fmt.Sprintf("%s/files/%s?exp=%d&sig=%s", l.BaseURL, key, exp, l.sign(key, exp)), nil
}

// Open verifies access (public prefix or valid signature) and opens the file.
func (l *Local) Open(key, exp, sig string) (*os.File, error) {
	if !strings.HasPrefix(key, "public/") {
		e, err := strconv.ParseInt(exp, 10, 64)
		if err != nil || time.Now().Unix() > e || !hmac.Equal([]byte(sig), []byte(l.sign(key, e))) {
			return nil, os.ErrPermission
		}
	}
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func New(driver, dir, baseURL, secret string) (*Local, error) {
	if driver != "local" {
		return nil, fmt.Errorf("storage driver %q is not built in: add an adapter implementing storage.Storage (S3, R2, Spaces)", driver)
	}
	return NewLocal(dir, baseURL, secret)
}
