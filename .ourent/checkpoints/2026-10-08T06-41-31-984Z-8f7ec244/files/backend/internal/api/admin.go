package api

import (
	"net/http"
	"strings"

	"kamolportfolio/backend/pkg/response"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) error {
	return s.one(w, r, "NOT_FOUND", `SELECT json_build_object(
'total_users',(SELECT count(*) FROM users),'total_products',(SELECT count(*) FROM products),'total_projects',(SELECT count(*) FROM projects),
'total_posts',(SELECT count(*) FROM posts),'total_orders',(SELECT count(*) FROM orders),
'total_revenue',(SELECT coalesce(sum(total),0) FROM orders WHERE status IN ('PAID','COMPLETED')),
'pending_orders',(SELECT count(*) FROM orders WHERE status='PENDING'),
'unread_messages',(SELECT count(*) FROM contact_messages WHERE status='NEW'),
'pending_comments',(SELECT count(*) FROM comments WHERE status='PENDING'),
'recent_orders',(SELECT coalesce(json_agg(t.x),'[]'::json) FROM (SELECT `+orderExpr+` AS x FROM orders o ORDER BY o.created_at DESC LIMIT 5) t),
'recent_comments',(SELECT coalesce(json_agg(t.x),'[]'::json) FROM (SELECT `+adminCommentExpr+` AS x FROM comments c JOIN users u ON u.id=c.user_id ORDER BY c.created_at DESC LIMIT 5) t),
'recent_products',(SELECT coalesce(json_agg(t.x),'[]'::json) FROM (SELECT `+productExpr(true)+` AS x FROM products p ORDER BY p.created_at DESC LIMIT 5) t),
'revenue_7d',(SELECT coalesce(json_agg(json_build_object('date',d::date,'total',coalesce(r.t,0)) ORDER BY d),'[]'::json)
  FROM generate_series(current_date-6,current_date,'1 day') d LEFT JOIN (SELECT created_at::date dt,sum(total) t FROM orders WHERE status IN ('PAID','COMPLETED') GROUP BY 1) r ON r.dt=d::date))`)
}

// ---------- orders ----------

func (s *Server) adminOrders(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	f := &filter{}
	if st := strings.ToUpper(q.Get("status")); st != "" {
		f.add("o.status = ?", st)
	}
	if sr := strings.TrimSpace(q.Get("search")); sr != "" {
		f.add("(o.order_number ILIKE ? OR o.customer_name ILIKE ? OR o.customer_email ILIKE ?)", "%"+escapeLike(sr)+"%")
	}
	return s.paged(w, r, orderExpr, "FROM orders o"+f.clause(), "o.created_at DESC, o.id DESC", f.args, 20)
}

func (s *Server) adminGetOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.one(w, r, "ORDER_NOT_FOUND", `SELECT `+orderExpr+` FROM orders o WHERE o.id=$1`, id)
}

func (s *Server) adminSetOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Status string }
	if err := decode(r, &in); err != nil {
		return err
	}
	st := strings.ToUpper(in.Status)
	switch st {
	case "PENDING", "PROCESSING", "PAID", "COMPLETED", "CANCELLED", "REFUNDED":
	default:
		return response.Validation(map[string]string{"status": "must be PENDING, PROCESSING, PAID, COMPLETED, CANCELLED or REFUNDED"})
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old string
	if err := tx.QueryRowContext(r.Context(), `SELECT status FROM orders WHERE id=$1 FOR UPDATE`, id).Scan(&old); err != nil {
		return response.E(404, "ORDER_NOT_FOUND", "Order not found.")
	}
	pay := map[string]string{"PAID": "PAID", "COMPLETED": "PAID", "REFUNDED": "REFUNDED"}[st]
	if _, err := tx.ExecContext(r.Context(), `UPDATE orders SET status=$1, payment_status=CASE WHEN $2<>'' THEN $2 ELSE payment_status END, updated_at=now() WHERE id=$3`, st, pay, id); err != nil {
		return err
	}
	switch st { // keep product access consistent with the order status
	case "PAID", "COMPLETED":
		if _, err := tx.ExecContext(r.Context(), `INSERT INTO product_access (user_id,product_id,order_id) SELECT o.user_id,i.product_id,o.id FROM orders o JOIN order_items i ON i.order_id=o.id
WHERE o.id=$1 AND o.user_id IS NOT NULL AND i.product_id IS NOT NULL ON CONFLICT (user_id,product_id) DO UPDATE SET status='ACTIVE', order_id=EXCLUDED.order_id`, id); err != nil {
			return err
		}
	case "CANCELLED", "REFUNDED":
		if _, err := tx.ExecContext(r.Context(), `UPDATE product_access SET status='REVOKED' WHERE order_id=$1`, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.audit(r, "order status changed", "order", id, map[string]any{"from": old, "to": st})
	return s.adminGetOrder(w, r)
}

// ---------- users ----------

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	f := &filter{}
	if st := strings.ToUpper(q.Get("status")); st != "" {
		f.add("u.status = ?", st)
	}
	if sr := strings.TrimSpace(q.Get("search")); sr != "" {
		f.add("(u.name ILIKE ? OR u.email ILIKE ?)", "%"+escapeLike(sr)+"%")
	}
	expr := `json_build_object('id',u.id,'name',u.name,'email',u.email,'avatar_url',u.avatar_url,'role',u.role,'status',u.status,'created_at',u.created_at,'last_login_at',u.last_login_at,'orders',(SELECT count(*) FROM orders o WHERE o.user_id=u.id))`
	return s.paged(w, r, expr, "FROM users u"+f.clause(), "u.created_at DESC, u.id DESC", f.args, 20)
}

func (s *Server) adminSetUser(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Status, Role *string }
	if err := decode(r, &in); err != nil {
		return err
	}
	if id == uid(r) {
		return response.E(409, "SELF_CHANGE", "You cannot change your own role or status.")
	}
	v := newV()
	if in.Status != nil {
		if *in.Status = strings.ToUpper(*in.Status); *in.Status != "ACTIVE" && *in.Status != "BLOCKED" {
			v.f["status"] = "must be ACTIVE or BLOCKED"
		}
	}
	if in.Role != nil {
		if *in.Role = strings.ToUpper(*in.Role); !map[string]bool{"USER": true, "ADMIN": true, "EDITOR": true, "MODERATOR": true}[*in.Role] {
			v.f["role"] = "must be USER, ADMIN, EDITOR or MODERATOR"
		}
	}
	if err := v.err(); err != nil {
		return err
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE users SET status=coalesce($1,status), role=coalesce($2,role), updated_at=now() WHERE id=$3`, in.Status, in.Role, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return response.E(404, "USER_NOT_FOUND", "User not found.")
	}
	if in.Status != nil && *in.Status == "BLOCKED" {
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id)
		s.audit(r, "user blocked", "user", id, nil)
	} else {
		s.audit(r, "user updated", "user", id, map[string]any{"status": in.Status, "role": in.Role})
	}
	return s.one(w, r, "USER_NOT_FOUND", `SELECT json_build_object('id',id,'name',name,'email',email,'role',role,'status',status) FROM users WHERE id=$1`, id)
}

// ---------- comment moderation ----------

const adminCommentExpr = `json_build_object('id',c.id,'type',CASE WHEN c.post_id IS NOT NULL THEN 'post' ELSE 'project' END,'target_id',coalesce(c.post_id,c.project_id),
'target_title',coalesce((SELECT title FROM posts WHERE id=c.post_id),(SELECT title FROM projects WHERE id=c.project_id)),'parent_id',c.parent_id,'user_id',c.user_id,'name',u.name,
'content',c.content,'status',c.status,'created_at',c.created_at)`

func (s *Server) adminComments(w http.ResponseWriter, r *http.Request) error {
	f := &filter{}
	if st := strings.ToUpper(r.URL.Query().Get("status")); st != "" {
		f.add("c.status = ?", st)
	}
	return s.paged(w, r, adminCommentExpr, "FROM comments c JOIN users u ON u.id=c.user_id"+f.clause(), "c.created_at DESC, c.id DESC", f.args, 50)
}

func (s *Server) setCommentStatus(w http.ResponseWriter, r *http.Request, id int64, st, action string) error {
	res, err := s.DB.ExecContext(r.Context(), `UPDATE comments SET status=$1, updated_at=now() WHERE id=$2`, st, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return response.E(404, "COMMENT_NOT_FOUND", "Comment not found.")
	}
	s.audit(r, action, "comment", id, nil)
	return s.one(w, r, "COMMENT_NOT_FOUND", `SELECT `+adminCommentExpr+` FROM comments c JOIN users u ON u.id=c.user_id WHERE c.id=$1`, id)
}

func (s *Server) adminSetComment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Status string }
	if err := decode(r, &in); err != nil {
		return err
	}
	st := strings.ToUpper(in.Status)
	if st != "VISIBLE" && st != "HIDDEN" && st != "PENDING" && st != "DELETED" {
		return response.Validation(map[string]string{"status": "must be VISIBLE, HIDDEN, PENDING or DELETED"})
	}
	return s.setCommentStatus(w, r, id, st, "comment "+strings.ToLower(st))
}

func (s *Server) adminDeleteComment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.setCommentStatus(w, r, id, "DELETED", "comment deleted")
}
