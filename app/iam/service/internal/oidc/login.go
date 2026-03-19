package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/helpers"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/redis"
)

// LoginHandler handles the OIDC login flow (GET redirects to SPA, POST handles form submission).
type LoginHandler struct {
	authnRepo  biz.AuthnRepo
	redis      *redis.Client
	log        *logger.Helper
	spaBaseURL string // base URL of the accounts SPA; TODO: make configurable via config file
}

// NewLoginHandler builds a handler that authenticates users and marks OIDC auth requests done.
func NewLoginHandler(authnRepo biz.AuthnRepo, rdb *redis.Client, l logger.Logger) *LoginHandler {
	return &LoginHandler{
		authnRepo:  authnRepo,
		redis:      rdb,
		log:        logger.NewHelper(l, logger.WithModule("oidc/login/iam-service")),
		spaBaseURL: "http://localhost:3001",
	}
}

// ServeHTTP dispatches to GET (render form) or POST (handle form submission).
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderLogin(w, r)
	case http.MethodPost:
		h.handleLogin(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LoginHandler) renderLogin(w http.ResponseWriter, r *http.Request) {
	authRequestID := r.URL.Query().Get("authRequestID")
	if authRequestID == "" {
		http.Error(w, "missing authRequestID", http.StatusBadRequest)
		return
	}
	spaLoginURL := fmt.Sprintf("%s/login?authRequestID=%s", h.spaBaseURL, url.QueryEscape(authRequestID))
	http.Redirect(w, r, spaLoginURL, http.StatusFound)
}

func (h *LoginHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	authRequestID := r.FormValue("authRequestID")
	email := r.FormValue("email")
	password := r.FormValue("password")

	callbackURL, err := h.authenticate(r.Context(), authRequestID, email, password)
	if err != nil {
		// Redirect back to the SPA login page with the error message.
		target := fmt.Sprintf("%s/login?authRequestID=%s&error=%s",
			h.spaBaseURL,
			url.QueryEscape(authRequestID),
			url.QueryEscape(err.Error()),
		)
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	http.Redirect(w, r, callbackURL, http.StatusFound)
}

// LoginCompleteHandler serves the JSON API at POST /login/complete.
type LoginCompleteHandler struct {
	lh *LoginHandler
}

// NewLoginCompleteHandler builds the API handler that returns callbackURL in JSON.
func NewLoginCompleteHandler(lh *LoginHandler) *LoginCompleteHandler {
	return &LoginCompleteHandler{lh: lh}
}

func (h *LoginCompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AuthRequestID string `json:"authRequestID"`
		Email         string `json:"email"`
		Password      string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	callbackURL, err := h.lh.authenticate(r.Context(), req.AuthRequestID, req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"callbackURL": callbackURL})
}

func (h *LoginHandler) authenticate(ctx context.Context, authRequestID, email, password string) (string, error) {
	if authRequestID == "" || email == "" || password == "" {
		return "", fmt.Errorf("missing required fields")
	}

	user, err := h.authnRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("invalid email or password")
		}
		h.log.Errorf("get user by email: %v", err)
		return "", fmt.Errorf("internal error")
	}

	if !helpers.BcryptCheck(password, user.Password) {
		return "", fmt.Errorf("invalid email or password")
	}

	key := "oidc:auth_request:" + authRequestID
	data, err := h.redis.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("auth request not found or expired")
	}

	var reqData map[string]any
	if err := json.Unmarshal([]byte(data), &reqData); err != nil {
		return "", fmt.Errorf("internal error")
	}
	reqData["user_id"] = user.ID
	reqData["auth_time"] = time.Now().UTC().Format(time.RFC3339Nano)
	reqData["done"] = true

	updated, _ := json.Marshal(reqData)
	if err := h.redis.Set(ctx, key, string(updated), 10*time.Minute); err != nil {
		return "", fmt.Errorf("internal error")
	}

	return fmt.Sprintf("/authorize/callback?id=%s", authRequestID), nil
}
