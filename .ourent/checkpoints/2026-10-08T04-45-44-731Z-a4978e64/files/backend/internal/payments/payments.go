// Package payments defines the provider abstraction. Only the development "test" provider is built in;
// a production provider (Stripe, Payme, Click, ...) is added by implementing Provider with its hosted/tokenized flow.
package payments

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Order struct {
	ID          int64
	Number      string
	AmountCents int64
	Currency    string
}

type Event struct {
	ID          string `json:"event_id"`
	Type        string `json:"type"` // payment.succeeded | payment.failed | payment.refunded
	PaymentRef  string `json:"payment_ref"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

var ErrSignature = errors.New("invalid webhook signature")
var ErrUnsupported = errors.New("not supported by this provider")

type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, o Order) (ref, checkoutURL string, err error)
	VerifyWebhook(h http.Header, body []byte) (Event, error)
	GetStatus(ctx context.Context, ref string) (string, error)
	Refund(ctx context.Context, ref string, amountCents int64) error
}

func New(name, secret string) (Provider, error) {
	if name == "test" {
		return &Test{Secret: secret}, nil
	}
	return nil, fmt.Errorf("payment provider %q is not built in: implement payments.Provider for it", name)
}

// Test is the development provider. It never moves money. Its "hosted page" is simulated by the
// backend endpoint POST /payments/test/complete, which signs an event and sends it through the real webhook path.
type Test struct{ Secret string }

func (t *Test) Name() string { return "test" }

func (t *Test) CreatePayment(_ context.Context, _ Order) (string, string, error) {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "test_" + hex.EncodeToString(b), "", nil
}

func (t *Test) Sign(body []byte) string {
	m := hmac.New(sha256.New, []byte(t.Secret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func (t *Test) VerifyWebhook(h http.Header, body []byte) (Event, error) {
	var e Event
	if !hmac.Equal([]byte(h.Get("X-Signature")), []byte(t.Sign(body))) {
		return e, ErrSignature
	}
	if err := json.Unmarshal(body, &e); err != nil || e.ID == "" || e.PaymentRef == "" {
		return e, errors.New("malformed event")
	}
	return e, nil
}

func (t *Test) GetStatus(context.Context, string) (string, error) { return "", ErrUnsupported }
func (t *Test) Refund(context.Context, string, int64) error       { return nil }
