package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"kamolportfolio/backend/pkg/response"
)

func nowUTC() time.Time { return time.Now().UTC() }

const commentExpr = `json_build_object('id',c.id,'parent_id',c.parent_id,'user_id',c.user_id,'name',u.name,'avatar_url',u.avatar_url,'content',c.content,'status',c.status,
'created_at',c.created_at,'updated_at',c.updated_at,
'like_count',(SELECT count(*) FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id),
'liked',EXISTS(SELECT 1 FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id AND l.user_id=$1))`

func (s *Server) targetOK(r *http.Request, kind string, id int64) error {
	table := map[string]string{"post": "posts", "project": "projects"}[kind]
	var x int
	err := s.DB.QueryRowContext(r.Context(), "SELECT 1 FROM "+table+" WHERE id=$1 AND published", id).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(404, strings.ToUpper(kind)+"_NOT_FOUND", "Not found.")
	}
	return err
}

func (s *Server) listComments(kind string) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		if err := s.targetOK(r, kind, id); err != nil {
			return err
		}
		b, err := s.raw(r.Context(), `SELECT coalesce(json_agg(`+commentExpr+` ORDER BY c.created_at,c.id),'[]'::json) FROM comments c JOIN users u ON u.id=c.user_id WHERE c.`+kind+`_id=$2 AND c.status='VISIBLE'`, uid(r), id)
		if err != nil {
			return err
		}
		response.OK(w, 200, b)
		return nil
	}
}

func (s *Server) insertComment(w http.ResponseWriter, r *http.Request, kind string, target int64, parent any, content string) error {
	v := newV()
	content = v.str("content", content, 1, 2000)
	if err := v.err(); err != nil {
		return err
	}
	var id int64
	err := s.DB.QueryRowContext(r.Context(), `INSERT INTO comments (user_id,`+kind+`_id,parent_id,content) VALUES ($1,$2,$3,$4) RETURNING id`, uid(r), target, parent, content).Scan(&id)
	if err != nil {
		return err
	}
	b, err := s.raw(r.Context(), `SELECT `+commentExpr+` FROM comments c JOIN users u ON u.id=c.user_id WHERE c.id=$2`, uid(r), id)
	if err != nil {
		return err
	}
	response.OK(w, 201, b)
	return nil
}

func (s *Server) addComment(kind string) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		var in struct {
			Content  string `json:"content"`
			ParentID *int64 `json:"parent_id"`
		}
		if err := decode(r, &in); err != nil {
			return err
		}
		if err := s.targetOK(r, kind, id); err != nil {
			return err
		}
		var parent any
		if in.ParentID != nil {
			var root sql.NullInt64
			err := s.DB.QueryRowContext(r.Context(), `SELECT coalesce(parent_id,id) FROM comments WHERE id=$1 AND `+kind+`_id=$2 AND status='VISIBLE'`, *in.ParentID, id).Scan(&root)
			if err != nil {
				return response.E(404, "COMMENT_NOT_FOUND", "Parent comment not found.")
			}
			parent = root.Int64
		}
		return s.insertComment(w, r, kind, id, parent, in.Content)
	}
}

// reply attaches to the thread root so threads stay one level deep.
func (s *Server) reply(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Content string }
	if err := decode(r, &in); err != nil {
		return err
	}
	var post, project, parent sql.NullInt64
	err = s.DB.QueryRowContext(r.Context(), `SELECT post_id,project_id,coalesce(parent_id,id) FROM comments WHERE id=$1 AND status='VISIBLE'`, id).Scan(&post, &project, &parent)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(404, "COMMENT_NOT_FOUND", "Comment not found.")
	}
	if err != nil {
		return err
	}
	if post.Valid {
		return s.insertComment(w, r, "post", post.Int64, parent.Int64, in.Content)
	}
	return s.insertComment(w, r, "project", project.Int64, parent.Int64, in.Content)
}

func (s *Server) ownComment(r *http.Request, id int64) error {
	var owner int64
	var status string
	err := s.DB.QueryRowContext(r.Context(), `SELECT user_id,status FROM comments WHERE id=$1`, id).Scan(&owner, &status)
	if errors.Is(err, sql.ErrNoRows) || status == "DELETED" {
		return response.E(404, "COMMENT_NOT_FOUND", "Comment not found.")
	}
	if err != nil {
		return err
	}
	if owner != uid(r) && ctxUser(r).Role != "ADMIN" {
		return response.E(403, "FORBIDDEN", "You can only change your own comments.")
	}
	return nil
}

func (s *Server) editComment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Content string }
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	in.Content = v.str("content", in.Content, 1, 2000)
	if err := v.err(); err != nil {
		return err
	}
	if err := s.ownComment(r, id); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE comments SET content=$1, updated_at=now() WHERE id=$2`, in.Content, id); err != nil {
		return err
	}
	return s.one(w, r, "COMMENT_NOT_FOUND", `SELECT `+commentExpr+` FROM comments c JOIN users u ON u.id=c.user_id WHERE c.id=$2`, uid(r), id)
}

func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.ownComment(r, id); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE comments SET status='DELETED', updated_at=now() WHERE id=$1 OR parent_id=$1`, id); err != nil {
		return err
	}
	response.NoContent(w)
	return nil
}

// like adds or removes the current user's like. The UNIQUE constraint makes repeated likes harmless.
func (s *Server) like(plural string, add bool) hfn {
	kind := map[string]string{"posts": "post", "projects": "project", "comments": "comment"}[plural]
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		var x int
		q := map[string]string{"post": `SELECT 1 FROM posts WHERE id=$1 AND published`, "project": `SELECT 1 FROM projects WHERE id=$1 AND published`, "comment": `SELECT 1 FROM comments WHERE id=$1 AND status='VISIBLE'`}[kind]
		if err := s.DB.QueryRowContext(r.Context(), q, id).Scan(&x); err != nil {
			return response.E(404, strings.ToUpper(kind)+"_NOT_FOUND", "Not found.")
		}
		if add {
			_, err = s.DB.ExecContext(r.Context(), `INSERT INTO likes (user_id,target_type,target_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, uid(r), kind, id)
		} else {
			_, err = s.DB.ExecContext(r.Context(), `DELETE FROM likes WHERE user_id=$1 AND target_type=$2 AND target_id=$3`, uid(r), kind, id)
		}
		if err != nil {
			return err
		}
		var n int
		if err := s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM likes WHERE target_type=$1 AND target_id=$2`, kind, id).Scan(&n); err != nil {
			return err
		}
		response.OK(w, 200, map[string]any{"liked": add, "like_count": n})
		return nil
	}
}

func (s *Server) favorite(add bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		if add {
			var x int
			if err := s.DB.QueryRowContext(r.Context(), `SELECT 1 FROM products WHERE id=$1 AND published`, id).Scan(&x); err != nil {
				return response.E(404, "PRODUCT_NOT_FOUND", "Product not found.")
			}
			_, err = s.DB.ExecContext(r.Context(), `INSERT INTO favorites (user_id,product_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, uid(r), id)
		} else {
			_, err = s.DB.ExecContext(r.Context(), `DELETE FROM favorites WHERE user_id=$1 AND product_id=$2`, uid(r), id)
		}
		if err != nil {
			return err
		}
		response.OK(w, 200, map[string]bool{"favorited": add})
		return nil
	}
}

func (s *Server) myFavorites(w http.ResponseWriter, r *http.Request) error {
	b, err := s.raw(r.Context(), `SELECT coalesce(json_agg(x.j),'[]'::json) FROM (SELECT `+productExpr(false)+` AS j FROM favorites f JOIN products p ON p.id=f.product_id WHERE f.user_id=$1 AND p.published ORDER BY f.created_at DESC) x`, uid(r))
	if err != nil {
		return err
	}
	response.OK(w, 200, b)
	return nil
}
