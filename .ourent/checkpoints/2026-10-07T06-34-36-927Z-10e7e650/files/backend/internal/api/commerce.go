package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"

	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/internal/payments"
	"kamolportfolio/backend/pkg/response"
)

// ---------- cart ----------

func (s *Server) cartID(ctx context.Context, user int64) (int64, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `INSERT INTO carts (user_id) VALUES ($1) ON CONFLICT (user_id) DO UPDATE SET updated_at=now() RETURNING id`, user).Scan(&id)
	return id, err
}

func (s *Server) getCart(w http.ResponseWriter, r *http.Request) error {
	return s.one(w, r, "CART_NOT_FOUND", `SELECT json_build_object('items',coalesce(json_agg(json_build_object('product_id',p.id,'quantity',i.quantity,'title',p.title,'slug',p.slug,'category',p.category,'price',p.price,'cover_image',p.cover_image,'line_total',p.price*i.quantity) ORDER BY p.id),'[]'::json),'subtotal',coalesce(sum(p.price*i.quantity),0),'currency','USD')
FROM carts c JOIN cart_items i ON i.cart_id=c.id JOIN products p ON p.id=i.product_id AND p.published WHERE c.user_id=$1`, uid(r))
}

func (s *Server) addCartItem(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Quantity == 0 {
		in.Quantity = 1
	}
	if in.ProductID < 1 || in.Quantity < 1 || in.Quantity > 99 {
		return response.Validation(map[string]string{"quantity": "must be 1-99"})
	}
	var x int
	if err := s.DB.QueryRowContext(r.Context(), `SELECT 1 FROM products WHERE id=$1 AND published`, in.ProductID).Scan(&x); err != nil {
		return response.E(404, "PRODUCT_NOT_FOUND", "Product not found.")
	}
	cid, err := s.cartID(r.Context(), uid(r))
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO cart_items (cart_id,product_id,quantity) VALUES ($1,$2,$3) ON CONFLICT (cart_id,product_id) DO UPDATE SET quantity=LEAST(99,cart_items.quantity+EXCLUDED.quantity)`, cid, in.ProductID, in.Quantity); err != nil {
		return err
	}
	return s.getCart(w, r)
}

func (s *Server) setCartItem(w http.ResponseWriter, r *http.Request) error {
	pid, err := pathID(r, "product_id")
	if err != nil {
		return err
	}
	var in struct{ Quantity int }
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Quantity < 0 || in.Quantity > 99 {
		return response.Validation(map[string]string{"quantity": "must be 0-99"})
	}
	cid, err := s.cartID(r.Context(), uid(r))
	if err != nil {
		return err
	}
	if in.Quantity == 0 {
		_, err = s.DB.ExecContext(r.Context(), `DELETE FROM cart_items WHERE cart_id=$1 AND product_id=$2`, cid, pid)
	} else {
		var res sql.Result
		res, err = s.DB.ExecContext(r.Context(), `UPDATE cart_items SET quantity=$3 WHERE cart_id=$1 AND product_id=$2`, cid, pid, in.Quantity)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				return response.E(404, "ITEM_NOT_FOUND", "Item is not in the cart.")
			}
		}
	}
	if err != nil {
		return err
	}
	return s.getCart(w, r)
}

func (s *Server) removeCartItem(w http.ResponseWriter, r *http.Request) error {
	pid, err := pathID(r, "product_id")
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), `DELETE FROM cart_items WHERE product_id=$2 AND cart_id=(SELECT id FROM carts WHERE user_id=$1)`, uid(r), pid); err != nil {
		return err
	}
	return s.getCart(w, r)
}

func (s *Server) clearCart(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.DB.ExecContext(r.Context(), `DELETE FROM cart_items WHERE cart_id=(SELECT id FROM carts WHERE user_id=$1)`, uid(r)); err != nil {
		return err
	}
	response.NoContent(w)
	return nil
}

// ---------- orders ----------

const orderExpr = `json_build_object('id',o.id,'order_number',o.order_number,'user_id',o.user_id,'customer_name',o.customer_name,'customer_email',o.customer_email,'customer_phone',o.customer_phone,
'country',o.country,'subtotal',o.subtotal,'discount',o.discount,'total',o.total,'currency',o.currency,'coupon_code',o.coupon_code,'status',o.status,'payment_status',o.payment_status,
'created_at',o.created_at,'updated_at',o.updated_at,
'items',(SELECT coalesce(json_agg(json_build_object('product_id',i.product_id,'title',i.title,'unit_price',i.unit_price,'quantity',i.quantity) ORDER BY i.id),'[]'::json) FROM order_items i WHERE i.order_id=o.id),
'payment',(SELECT json_build_object('id',pm.id,'status',pm.status,'provider',pm.provider) FROM payments pm WHERE pm.order_id=o.id ORDER BY pm.id DESC LIMIT 1))`

var phoneRe = regexp.MustCompile(`^[0-9+()\-\s]{0,30}$`)

type orderItemIn struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

func orderNumber() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("KA-%s-%06d", time.Now().UTC().Format("060102"), n.Int64())
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Customer struct{ Name, Email, Phone, Country string } `json:"customer"`
		Items    []orderItemIn                                `json:"items"`
		Coupon   string                                       `json:"coupon"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	name := v.str("customer.name", in.Customer.Name, 1, 100)
	email := v.email("customer.email", in.Customer.Email)
	phone := strings.TrimSpace(in.Customer.Phone)
	if !phoneRe.MatchString(phone) {
		v.f["customer.phone"] = "invalid phone number"
	}
	country := v.str("customer.country", in.Customer.Country, 0, 80)
	if err := v.err(); err != nil {
		return err
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(idem) > 100 {
		return response.Validation(map[string]string{"Idempotency-Key": "too long"})
	}
	if idem != "" { // retried request: return the order created by the first attempt
		var oid int64
		if err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM orders WHERE user_id=$1 AND idempotency_key=$2`, uid(r), idem).Scan(&oid); err == nil {
			b, err := s.raw(r.Context(), `SELECT `+orderExpr+` FROM orders o WHERE o.id=$1`, oid)
			if err != nil {
				return err
			}
			response.OK(w, 200, b)
			return nil
		}
	}
	items := in.Items
	if len(items) == 0 { // fall back to the server-side cart
		rows, err := s.DB.QueryContext(r.Context(), `SELECT i.product_id,i.quantity FROM carts c JOIN cart_items i ON i.cart_id=c.id WHERE c.user_id=$1`, uid(r))
		if err != nil {
			return err
		}
		for rows.Next() {
			var it orderItemIn
			if err := rows.Scan(&it.ProductID, &it.Quantity); err != nil {
				rows.Close()
				return err
			}
			items = append(items, it)
		}
		rows.Close()
	}
	if len(items) == 0 || len(items) > 50 {
		return response.E(422, "EMPTY_ORDER", "Your order has no items.")
	}
	merged, order := map[int64]int{}, []int64{}
	for _, it := range items {
		if it.ProductID < 1 || it.Quantity < 1 || it.Quantity > 99 {
			return response.Validation(map[string]string{"items": "each item needs a valid product_id and a quantity of 1-99"})
		}
		if _, ok := merged[it.ProductID]; !ok {
			order = append(order, it.ProductID)
		}
		merged[it.ProductID] = min(99, merged[it.ProductID]+it.Quantity)
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type line struct {
		pid   int64
		title string
		cents int64
		qty   int
	}
	var lines []line
	var subtotal int64
	currency := ""
	for _, pid := range order { // prices always come from the database
		var l line
		var cur string
		var pub bool
		err := tx.QueryRowContext(r.Context(), `SELECT title,(price*100)::bigint,currency,published FROM products WHERE id=$1 FOR SHARE`, pid).Scan(&l.title, &l.cents, &cur, &pub)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !pub) {
			return response.E(422, "PRODUCT_UNAVAILABLE", fmt.Sprintf("Product %d is not available.", pid))
		}
		if err != nil {
			return err
		}
		if currency == "" {
			currency = cur
		} else if currency != cur {
			return response.E(422, "MIXED_CURRENCY", "Products with different currencies cannot be ordered together.")
		}
		l.pid, l.qty = pid, merged[pid]
		subtotal += l.cents * int64(l.qty)
		lines = append(lines, l)
	}
	var discount int64
	var coupon any
	if c := strings.TrimSpace(in.Coupon); c != "" {
		var pct int
		err := tx.QueryRowContext(r.Context(), `SELECT percent FROM coupons WHERE upper(code)=upper($1) AND active AND (expires_at IS NULL OR expires_at>now())`, c).Scan(&pct)
		if err != nil {
			return response.E(422, "INVALID_COUPON", "This coupon is invalid or has expired.")
		}
		discount, coupon = subtotal*int64(pct)/100, strings.ToUpper(c)
	}
	money := func(c int64) string { return fmt.Sprintf("%d.%02d", c/100, c%100) }
	var oid int64
	var idemArg any
	if idem != "" {
		idemArg = idem
	}
	for try := 0; ; try++ {
		err = tx.QueryRowContext(r.Context(), `INSERT INTO orders (order_number,user_id,customer_name,customer_email,customer_phone,country,subtotal,discount,total,currency,coupon_code,idempotency_key)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, orderNumber(), uid(r), name, email, phone, country, money(subtotal), money(discount), money(subtotal-discount), currency, coupon, idemArg).Scan(&oid)
		var pe *pq.Error
		if errors.As(err, &pe) && pe.Code == "23505" && strings.Contains(pe.Constraint, "order_number") && try < 5 {
			continue
		}
		break
	}
	if err != nil {
		var pe *pq.Error
		if errors.As(err, &pe) && pe.Code == "23505" && idem != "" { // concurrent duplicate of the same request
			return response.E(409, "DUPLICATE_REQUEST", "This order is already being processed.")
		}
		return err
	}
	for _, l := range lines {
		if _, err := tx.ExecContext(r.Context(), `INSERT INTO order_items (order_id,product_id,title,unit_price,quantity) VALUES ($1,$2,$3,$4,$5)`, oid, l.pid, l.title, money(l.cents), l.qty); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	b, err := s.raw(r.Context(), `SELECT `+orderExpr+` FROM orders o WHERE o.id=$1`, oid)
	if err != nil {
		return err
	}
	response.OK(w, 201, b)
	return nil
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.one(w, r, "ORDER_NOT_FOUND", `SELECT `+orderExpr+` FROM orders o WHERE o.id=$1 AND (o.user_id=$2 OR $3::boolean)`, id, uid(r), ctxUser(r).Role == "ADMIN")
}

func (s *Server) myOrders(w http.ResponseWriter, r *http.Request) error {
	return s.paged(w, r, orderExpr, "FROM orders o WHERE o.user_id=$1", "o.created_at DESC, o.id DESC", []any{uid(r)}, 20)
}

func (s *Server) myPurchases(w http.ResponseWriter, r *http.Request) error {
	b, err := s.raw(r.Context(), `SELECT coalesce(json_agg(x.j),'[]'::json) FROM (SELECT json_build_object('product',`+productExpr(false)+`,'granted_at',a.granted_at,'expires_at',a.expires_at,'order_id',a.order_id) AS j
FROM product_access a JOIN products p ON p.id=a.product_id WHERE a.user_id=$1 AND a.status='ACTIVE' AND (a.expires_at IS NULL OR a.expires_at>now()) ORDER BY a.granted_at DESC) x`, uid(r))
	if err != nil {
		return err
	}
	response.OK(w, 200, b)
	return nil
}

// ---------- payments ----------

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		OrderID int64 `json:"order_id"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var o payments.Order
	var status, pstatus string
	err = tx.QueryRowContext(r.Context(), `SELECT id,order_number,(total*100)::bigint,currency,status,payment_status FROM orders WHERE id=$1 AND user_id=$2 FOR UPDATE`, in.OrderID, uid(r)).Scan(&o.ID, &o.Number, &o.AmountCents, &o.Currency, &status, &pstatus)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(404, "ORDER_NOT_FOUND", "Order not found.")
	}
	if err != nil {
		return err
	}
	if status != "PENDING" || pstatus == "PAID" {
		return response.E(409, "ORDER_NOT_PAYABLE", "This order cannot be paid.")
	}
	var existing int64
	if err := tx.QueryRowContext(r.Context(), `SELECT id FROM payments WHERE order_id=$1 AND status='PENDING' ORDER BY id DESC LIMIT 1`, o.ID).Scan(&existing); err == nil {
		tx.Rollback()
		return s.one(w, r, "PAYMENT_NOT_FOUND", paymentSQL, existing, uid(r))
	}
	ref, url, err := s.Pay.CreatePayment(r.Context(), o)
	if err != nil {
		s.Log.Error("payment provider error", "err", err.Error())
		return response.E(502, "PROVIDER_ERROR", "The payment provider is unavailable. Please try again.")
	}
	var pid int64
	if err := tx.QueryRowContext(r.Context(), `INSERT INTO payments (order_id,provider,provider_ref,amount,currency,checkout_url) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		o.ID, s.Pay.Name(), ref, fmt.Sprintf("%d.%02d", o.AmountCents/100, o.AmountCents%100), o.Currency, url).Scan(&pid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE orders SET payment_status='PENDING', updated_at=now() WHERE id=$1`, o.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	b, err := s.raw(r.Context(), paymentSQL, pid, uid(r))
	if err != nil {
		return err
	}
	response.OK(w, 201, b)
	return nil
}

const paymentSQL = `SELECT json_build_object('payment_id',pm.id,'order_id',pm.order_id,'provider',pm.provider,'status',pm.status,'checkout_url',pm.checkout_url,'amount',pm.amount,'currency',pm.currency,'created_at',pm.created_at)
FROM payments pm JOIN orders o ON o.id=pm.order_id WHERE pm.id=$1 AND o.user_id=$2`

func (s *Server) getPayment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.one(w, r, "PAYMENT_NOT_FOUND", paymentSQL, id, uid(r))
}

// webhook is the ONLY path that can mark a payment as paid. The frontend is never trusted.
func (s *Server) webhook(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		return response.E(400, "BAD_BODY", "Invalid request body.")
	}
	dup, err := s.processWebhook(r, body)
	if err != nil {
		return err
	}
	response.OK(w, 200, map[string]bool{"received": true, "duplicate": dup})
	return nil
}

func (s *Server) processWebhook(r *http.Request, body []byte) (bool, error) {
	ev, err := s.Pay.VerifyWebhook(r.Header, body)
	if errors.Is(err, payments.ErrSignature) {
		return false, response.E(401, "INVALID_SIGNATURE", "Invalid webhook signature.")
	}
	if err != nil {
		return false, response.E(400, "BAD_EVENT", "Malformed webhook event.")
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO payment_events (event_id,type,payload) VALUES ($1,$2,$3) ON CONFLICT (event_id) DO NOTHING`, ev.ID, ev.Type, string(body))
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return true, nil // already processed
	}
	var pid, orderID int64
	var amount int64
	var cur, pstatus string
	err = tx.QueryRowContext(ctx, `SELECT id,order_id,(amount*100)::bigint,currency,status FROM payments WHERE provider_ref=$1 FOR UPDATE`, ev.PaymentRef).Scan(&pid, &orderID, &amount, &cur, &pstatus)
	if errors.Is(err, sql.ErrNoRows) {
		return false, response.E(404, "PAYMENT_NOT_FOUND", "Unknown payment.")
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payment_events SET payment_id=$1 WHERE event_id=$2`, pid, ev.ID); err != nil {
		return false, err
	}
	switch ev.Type {
	case "payment.succeeded":
		if pstatus != "PENDING" {
			break
		}
		if ev.AmountCents != amount || ev.Currency != cur {
			s.Log.Warn("webhook amount mismatch", "payment_id", pid)
			return false, response.E(422, "AMOUNT_MISMATCH", "Amount does not match the order.")
		}
		stmts := []string{
			`UPDATE payments SET status='PAID', paid_at=now(), updated_at=now() WHERE id=$1`,
			`UPDATE orders SET payment_status='PAID', status='PAID', updated_at=now() WHERE id=(SELECT order_id FROM payments WHERE id=$1) AND status IN ('PENDING','PROCESSING')`,
			`INSERT INTO product_access (user_id,product_id,order_id) SELECT o.user_id,i.product_id,o.id FROM payments pm JOIN orders o ON o.id=pm.order_id JOIN order_items i ON i.order_id=o.id
 WHERE pm.id=$1 AND o.user_id IS NOT NULL AND i.product_id IS NOT NULL ON CONFLICT (user_id,product_id) DO UPDATE SET status='ACTIVE', order_id=EXCLUDED.order_id, granted_at=now()`,
			`DELETE FROM cart_items WHERE cart_id=(SELECT c.id FROM carts c JOIN orders o ON o.user_id=c.user_id JOIN payments pm ON pm.order_id=o.id WHERE pm.id=$1)
 AND product_id IN (SELECT i.product_id FROM order_items i JOIN payments pm ON pm.order_id=i.order_id WHERE pm.id=$1)`,
		}
		for _, q := range stmts {
			if _, err := tx.ExecContext(ctx, q, pid); err != nil {
				return false, err
			}
		}
	case "payment.failed":
		if pstatus == "PENDING" {
			if _, err := tx.ExecContext(ctx, `UPDATE payments SET status='FAILED', updated_at=now() WHERE id=$1`, pid); err != nil {
				return false, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE orders SET payment_status='FAILED', updated_at=now() WHERE id=$1 AND status='PENDING'`, orderID); err != nil {
				return false, err
			}
		}
	case "payment.refunded":
		if pstatus == "PAID" {
			for _, q := range []string{
				`UPDATE payments SET status='REFUNDED', updated_at=now() WHERE id=$1`,
				`UPDATE orders SET status='REFUNDED', payment_status='REFUNDED', updated_at=now() WHERE id=$2`,
				`UPDATE product_access SET status='REVOKED' WHERE order_id=$2`,
			} {
				if _, err := tx.ExecContext(ctx, q, pid, orderID); err != nil {
					return false, err
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	s.audit(r, ev.Type, "payment", pid, map[string]any{"event_id": ev.ID})
	return false, nil
}

// testComplete simulates the provider's hosted payment page (development only). It signs an event and
// pushes it through the same verified webhook path, so the browser never marks anything as paid itself.
func (s *Server) testComplete(w http.ResponseWriter, r *http.Request) error {
	tp, ok := s.Pay.(*payments.Test)
	if !ok || s.Cfg.IsProd() {
		return response.E(404, "NOT_FOUND", "Not found.")
	}
	var in struct {
		PaymentID int64  `json:"payment_id"`
		Outcome   string `json:"outcome"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	var ref, cur string
	var cents int64
	err := s.DB.QueryRowContext(r.Context(), `SELECT pm.provider_ref,(pm.amount*100)::bigint,pm.currency FROM payments pm JOIN orders o ON o.id=pm.order_id WHERE pm.id=$1 AND o.user_id=$2 AND pm.provider='test'`, in.PaymentID, uid(r)).Scan(&ref, &cents, &cur)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(404, "PAYMENT_NOT_FOUND", "Payment not found.")
	}
	if err != nil {
		return err
	}
	typ := "payment.succeeded"
	if in.Outcome == "fail" {
		typ = "payment.failed"
	}
	body := fmt.Sprintf(`{"event_id":"evt_%s","type":"%s","payment_ref":"%s","amount_cents":%d,"currency":"%s"}`, auth.RandomToken(9), typ, ref, cents, cur)
	r2 := r.Clone(r.Context())
	r2.Header = http.Header{"X-Signature": []string{tp.Sign([]byte(body))}}
	if _, err := s.processWebhook(r2, []byte(body)); err != nil {
		return err
	}
	return s.one(w, r, "PAYMENT_NOT_FOUND", paymentSQL, in.PaymentID, uid(r))
}

// ---------- protected product access ----------

func (s *Server) accessURL(r *http.Request, id int64) (string, bool, error) {
	var key string
	var hasAccess bool
	err := s.DB.QueryRowContext(r.Context(), `SELECT p.download_url, EXISTS(SELECT 1 FROM product_access a WHERE a.product_id=p.id AND a.user_id=$2 AND a.status='ACTIVE' AND (a.expires_at IS NULL OR a.expires_at>now()))
FROM products p WHERE p.id=$1`, id, uid(r)).Scan(&key, &hasAccess)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, response.E(404, "PRODUCT_NOT_FOUND", "Product not found.")
	}
	if err != nil {
		return "", false, err
	}
	if !hasAccess && ctxUser(r).Role != "ADMIN" {
		return "", false, response.E(403, "NO_ACCESS", "Purchase this product to get access.")
	}
	if key == "" {
		return "", true, nil
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return key, true, nil
	}
	u, err := s.Store.GetURL(key, 5*time.Minute)
	return u, true, err
}

func (s *Server) productAccess(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	u, ok, err := s.accessURL(r, id)
	if err != nil {
		return err
	}
	response.OK(w, 200, map[string]any{"has_access": ok, "has_download": u != "", "download_url": u})
	return nil
}

func (s *Server) productDownload(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	u, _, err := s.accessURL(r, id)
	if err != nil {
		return err
	}
	if u == "" {
		return response.E(404, "NO_DOWNLOAD", "No downloadable file is connected to this product yet.")
	}
	http.Redirect(w, r, u, http.StatusFound)
	return nil
}
