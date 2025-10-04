package rest

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

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
		writeJSON(w, http.StatusBadRequest, struct {
			Message string `json:"message"`
			Cause   string `json:"cause"`
		}{"Account creation failed", "Required user_id and password"})
		return
	}

	user, err := s.UC.SignUp(req.UserID, req.Password)
	if err != nil {
		switch e := err.(type) {
		case *usecase.ValidationError:
			cause := validationCause(e.Reason)
			writeJSON(w, http.StatusBadRequest, struct {
				Message string `json:"message"`
				Cause   string `json:"cause"`
			}{"Account creation failed", cause})
			return
		default:
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
			if errors.Is(err, usecase.ErrNotFound) {
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
			case *usecase.ValidationError:
				cause := validationCause(e.Reason)
				// usecase が返す理由コードを HTTP 応答用メッセージに変換
				writeJSON(w, http.StatusBadRequest, struct {
					Message string `json:"message"`
					Cause   string `json:"cause"`
				}{"User updation failed", cause})
				return
			default:
				if errors.Is(err, usecase.ErrNotFound) {
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

func validationCause(reason usecase.ValidationReason) string {
	switch reason {
	case usecase.ValidationReasonCredentialRequired:
		return "Required user_id and password"
	case usecase.ValidationReasonInputLength:
		return "Input length is incorrect"
	case usecase.ValidationReasonInvalidPattern:
		return "Incorrect character pattern"
	case usecase.ValidationReasonProfileRequired:
		return "Required nickname or comment"
	case usecase.ValidationReasonProfileConstraint:
		return "String length limit exceeded or containing invalid characters"
	case usecase.ValidationReasonUserAlreadyExists:
		return "Already same user_id is used"
	case usecase.ValidationReasonNotUpdatableIDOrPass:
		return "Not updatable user_id and password"
	default:
		return "Validation failed"
	}
}
