package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"accountapi/internal/domain"
	"accountapi/internal/infrastructure/repository/memrepo"
	"accountapi/internal/usecase"
)

type Server struct {
	UC  *usecase.Usecase
	mux *http.ServeMux
}

func New() *Server {
	repo := memrepo.New()
	uc := &usecase.Usecase{Repo: repo}
	if Env("SEED_TEST_USER", "true") == "true" {
		seedIfNeeded(uc)
	}
	s := &Server{UC: uc, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.healthz)
	s.mux.HandleFunc("/signup", s.handleSignup)
	s.mux.HandleFunc("/users/", s.handleUsers) // /users/{user_id}
	s.mux.HandleFunc("/close", s.handleClose)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// アクセスログ（簡易）
	defer func() {
		log.Printf("%s %s %dms UA=%q", r.Method, r.URL.Path, time.Since(start).Milliseconds(), r.UserAgent())
	}()
	s.mux.ServeHTTP(w, r)
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, messageOnly{Message: "ok"})
}

// POST /signup
func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MiB
	defer r.Body.Close()

	var req signUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body が壊れている場合もフォーマット的には 400 の cause を合わせる
		writeJSON(w, http.StatusBadRequest, struct {
			Message string `json:"message"`
			Cause   string `json:"cause"`
		}{"Account creation failed", "Required user_id and password"})
		return
	}

	user, err := s.UC.SignUp(req.UserID, req.Password)
	if err != nil {
		switch e := err.(type) {
		case *domain.ErrValidation:
			writeJSON(w, http.StatusBadRequest, struct {
				Message string `json:"message"`
				Cause   string `json:"cause"`
			}{"Account creation failed", e.Cause})
			return
		default:
			if errors.Is(err, domain.ErrAlreadyExists) {
				writeJSON(w, http.StatusBadRequest, struct {
					Message string `json:"message"`
					Cause   string `json:"cause"`
				}{"Account creation failed", "Already same user_id is used"})
				return
			}
			// サーバ内部エラー
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// nickname 未設定時は user_id と同じ値を返す
	resp := signUpResponse{
		Message: "Account successfully created",
		User:    userSummaryNoComm{UserID: user.UserID, Nickname: user.UserID},
	}
	writeJSON(w, http.StatusOK, resp)
}

// /users/{user_id}
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == http.MethodGet || r.Method == http.MethodPatch) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/users/"), "/")
	if len(parts) != 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	pathUserID := parts[0]

	authUser, authPass, ok := r.BasicAuth()
	if !ok {
		writeAuthFailed(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		u, err := s.UC.GetUser(pathUserID, authUser, authPass)
		if err != nil {
			if errors.Is(err, usecase.ErrAuthFailed) {
				writeAuthFailed(w)
				return
			}
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, messageOnly{Message: "No user found"})
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		// nickname 未設定なら user_id と同値
		nn := u.Nickname
		if nn == "" {
			nn = u.UserID
		}
		var commentPtr *string
		if u.Comment != "" {
			c := u.Comment
			commentPtr = &c
		}
		writeJSON(w, http.StatusOK, userResponse{
			Message: "User details by user_id",
			User:    userDetail{UserID: u.UserID, Nickname: nn, Comment: commentPtr},
		})
	case http.MethodPatch:
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		var req updateUserRequest
		if err := dec.Decode(&req); err != nil {
			// JSON 壊れているなど → 仕様上は「Required nickname or comment」を優先
			writeJSON(w, http.StatusBadRequest, struct {
				Message string `json:"message"`
				Cause   string `json:"cause"`
			}{"User updation failed", "Required nickname or comment"})
			return
		}
		// user_id/password が body に含まれるだけで NG
		forbid := (req.UserID != nil) || (req.Password != nil)

		u, err := s.UC.UpdateUser(pathUserID, authUser, authPass, req.Nickname, req.Comment, forbid)
		if err != nil {
			if errors.Is(err, usecase.ErrNoPerm) {
				// 403
				writeJSON(w, http.StatusForbidden, struct {
					Message string `json:"message"`
				}{"No permission for update"})
				return
			}
			if errors.Is(err, usecase.ErrAuthFailed) {
				writeAuthFailed(w)
				return
			}
			switch e := err.(type) {
			case *domain.ErrValidation:
				// cause は 2 種類のいずれか（UC からそのまま）
				writeJSON(w, http.StatusBadRequest, struct {
					Message string `json:"message"`
					Cause   string `json:"cause"`
				}{"User updation failed", e.Cause})
				return
			default:
				if errors.Is(err, domain.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, messageOnly{Message: "No user found"})
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
		// 応答生成（nickname 未設定は user_id）
		nn := u.Nickname
		if nn == "" {
			nn = u.UserID
		}
		var commentPtr *string
		if u.Comment != "" {
			c := u.Comment
			commentPtr = &c
		}
		writeJSON(w, http.StatusOK, userResponse{
			Message: "User successfully updated",
			User:    userDetail{UserID: u.UserID, Nickname: nn, Comment: commentPtr},
		})
	}
}

// POST /close
func (s *Server) handleClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	authUser, authPass, ok := r.BasicAuth()
	if !ok {
		writeAuthFailed(w)
		return
	}
	if err := s.UC.CloseUser(authUser, authPass); err != nil {
		// /close は未存在も 401
		writeAuthFailed(w)
		return
	}
	writeJSON(w, http.StatusOK, messageOnly{Message: "Account and user successfully removed"})
}

// ---- 起動ユーティリティ ----

func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func seedIfNeeded(uc *usecase.Usecase) {
	if _, err := uc.GetUser("TaroYamada", "TaroYamada", "PaSSwd4TY"); err == nil {
		return
	}
	user, err := uc.SignUp("TaroYamada", "PaSSwd4TY")
	if err != nil {
		return
	}
	nn := strPtr("たろー")
	cm := strPtr("僕は元気です")
	_, _ = uc.UpdateUser(user.UserID, user.UserID, "PaSSwd4TY", nn, cm, false)
}

func strPtr(s string) *string {
	return &s
}
