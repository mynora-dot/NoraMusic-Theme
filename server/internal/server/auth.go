package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mynora/noramusic-theme-server/internal/store"
	"golang.org/x/crypto/argon2"
)

type principal struct{ ID, Email, SessionID, CSRF string }

func (s *Server) authNetwork(w http.ResponseWriter, r *http.Request) bool {
	if !s.allowedAdminNetwork(r) {
		s.jsonError(w, r, http.StatusForbidden, "ADMIN_NETWORK_DENIED")
		return false
	}
	if (s.Config.Mode == "public" || (s.Config.Mode == "lan" && !s.Config.AllowLANHTTP)) && !s.secureRequest(r) {
		s.jsonError(w, r, http.StatusForbidden, "HTTPS_REQUIRED")
		return false
	}
	return true
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func hashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, 2, 64*1024, 2, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + "." + base64.RawStdEncoding.EncodeToString(key)
}
func validPassword(encoded, password string) bool {
	p := strings.Split(encoded, ".")
	if len(p) != 2 {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(p[0])
	want, e2 := base64.RawStdEncoding.DecodeString(p[1])
	if e1 != nil || e2 != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 2, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(got, want) == 1
}
func tokenHash(v string) string { h := digest(v); return hex.EncodeToString(h[:]) }
func (s *Server) adminFromRequest(r *http.Request) (principal, bool) {
	if s.Config.Mode == "lan" && s.Config.AllowInsecureAdmin {
		var p principal
		if err := s.Store.DB.QueryRow(`SELECT id,email FROM admins ORDER BY created_at LIMIT 1`).Scan(&p.ID, &p.Email); err == nil {
			return p, true
		}
	}
	if c, err := r.Cookie("nora_admin"); err == nil && c.Value != "" {
		var p principal
		var expires int64
		err := s.Store.DB.QueryRow(`SELECT a.id,a.email,x.id,x.csrf_token,x.expires_at FROM sessions x JOIN admins a ON a.id=x.admin_id WHERE x.token_hash=?`, tokenHash(c.Value)).Scan(&p.ID, &p.Email, &p.SessionID, &p.CSRF, &expires)
		if err == nil && expires > store.Now() {
			return p, true
		}
	}
	if raw := r.Header.Get("Authorization"); strings.HasPrefix(raw, "Bearer ") {
		var p principal
		var expires int64
		err := s.Store.DB.QueryRow(`SELECT a.id,a.email,'','',0 FROM api_tokens t JOIN admins a ON a.id=t.admin_id WHERE t.token_hash=? AND t.revoked_at=0`, tokenHash(strings.TrimSpace(strings.TrimPrefix(raw, "Bearer ")))).Scan(&p.ID, &p.Email, &p.SessionID, &p.CSRF, &expires)
		if err == nil {
			return p, true
		}
	}
	return principal{}, false
}
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request, write bool) (principal, bool) {
	p, ok := s.adminFromRequest(r)
	if !ok {
		s.jsonError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return principal{}, false
	}
	if write && r.Method != http.MethodGet && r.Method != http.MethodHead && p.CSRF != "" {
		if subtle.ConstantTimeCompare([]byte(p.CSRF), []byte(r.Header.Get("X-CSRF-Token"))) != 1 {
			s.jsonError(w, r, http.StatusForbidden, "CSRF_REQUIRED")
			return principal{}, false
		}
	}
	return p, true
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	if !s.allowAuth(r, "login") {
		s.jsonError(w, r, http.StatusTooManyRequests, "RATE_LIMITED")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var id, hash string
	err := s.Store.DB.QueryRow(`SELECT id,password_hash FROM admins WHERE email=?`, email).Scan(&id, &hash)
	if err != nil || !validPassword(hash, in.Password) {
		s.jsonError(w, r, http.StatusUnauthorized, "AUTH_INVALID")
		return
	}
	token, csrf := randomToken(32), randomToken(24)
	expires := time.Now().Add(12 * time.Hour).UnixMilli()
	_, err = s.Store.DB.Exec(`INSERT INTO sessions(id,admin_id,token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?,?,?)`, store.ID("ses_"), id, tokenHash(token), csrf, expires, store.Now())
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	secure := s.Config.Mode == "public"
	http.SetCookie(w, &http.Cookie{Name: "nora_admin", Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
	s.json(w, r, 200, map[string]any{"email": email, "csrfToken": csrf, "expiresAt": time.UnixMilli(expires).UTC().Format(time.RFC3339)})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	if p, ok := s.adminFromRequest(r); ok && p.SessionID != "" {
		_, _ = s.Store.DB.Exec(`DELETE FROM sessions WHERE id=?`, p.SessionID)
	}
	http.SetCookie(w, &http.Cookie{Name: "nora_admin", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	s.json(w, r, 200, map[string]bool{"loggedOut": true})
}
func totpCode(secret string, t time.Time) string {
	raw, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	counter := uint64(t.Unix() / 30)
	msg := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		msg[i] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, raw)
	_, _ = mac.Write(msg)
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 15
	n := uint32(sum[off]&127)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return fmt.Sprintf("%06d", n%1000000)
}
func validTOTP(secret, code string) bool {
	for d := -1; d <= 1; d++ {
		if subtle.ConstantTimeCompare([]byte(totpCode(secret, time.Now().Add(time.Duration(d)*30*time.Second))), []byte(code)) == 1 {
			return true
		}
	}
	return false
}
func newTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")
}
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	var n int
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	s.json(w, r, 200, map[string]any{"setupRequired": n == 0, "mode": s.Config.Mode, "mfaRequired": s.Config.Mode == "public"})
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	if !s.allowAuth(r, "register") {
		s.jsonError(w, r, http.StatusTooManyRequests, "RATE_LIMITED")
		return
	}
	var n int
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	if n > 0 {
		s.jsonError(w, r, 409, "ADMIN_ALREADY_INITIALIZED")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	if len(in.Password) < 12 || len(in.Password) > 256 {
		s.jsonError(w, r, 400, "PASSWORD_WEAK")
		return
	}
	id := store.ID("adm_")
	email := strings.ToLower(strings.TrimSpace(in.Email))
	_, err := s.Store.DB.Exec(`INSERT INTO admins(id,email,password_hash,created_at) VALUES(?,?,?,?)`, id, email, hashPassword(in.Password), store.Now())
	if err != nil {
		s.jsonError(w, r, 400, "INVALID_ADMIN")
		return
	}
	// Issue a session directly because the registration request body has already
	// been consumed; calling login here would attempt to decode an empty body.
	token, csrf := randomToken(32), randomToken(24)
	expires := time.Now().Add(12 * time.Hour).UnixMilli()
	if _, err = s.Store.DB.Exec(`INSERT INTO sessions(id,admin_id,token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?,?,?)`, store.ID("ses_"), id, tokenHash(token), csrf, expires, store.Now()); err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "nora_admin", Value: token, Path: "/", HttpOnly: true, Secure: s.Config.Mode == "public", SameSite: http.SameSiteStrictMode, MaxAge: 43200})
	s.json(w, r, 201, map[string]any{"email": email, "csrfToken": csrf, "expiresAt": time.UnixMilli(expires).UTC().Format(time.RFC3339)})
}
func (s *Server) setupMFA(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	p, ok := s.requireAdmin(w, r, true)
	if !ok {
		return
	}
	secret := newTOTPSecret()
	_, err := s.Store.DB.Exec(`UPDATE admins SET totp_secret=? WHERE id=?`, secret, p.ID)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	uri := fmt.Sprintf("otpauth://totp/NoraMusic:%s?secret=%s&issuer=NoraMusic", p.Email, secret)
	s.json(w, r, 200, map[string]any{"secret": secret, "otpauthUri": uri})
}
func (s *Server) verifyMFA(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	p, ok := s.requireAdmin(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	var secret string
	if s.Store.DB.QueryRow(`SELECT totp_secret FROM admins WHERE id=?`, p.ID).Scan(&secret) != nil || !validTOTP(secret, in.Code) {
		s.jsonError(w, r, 400, "MFA_INVALID")
		return
	}
	_, _ = s.Store.DB.Exec(`UPDATE admins SET totp_enabled=1 WHERE id=?`, p.ID)
	s.json(w, r, 200, map[string]bool{"enabled": true})
}

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, false)
	if !ok {
		return
	}
	rows, err := s.Store.DB.Query(`SELECT id,name,created_at,revoked_at FROM api_tokens WHERE admin_id=? ORDER BY created_at DESC`, p.ID)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name string
		var created, revoked int64
		_ = rows.Scan(&id, &name, &created, &revoked)
		items = append(items, map[string]any{"id": id, "name": name, "createdAt": time.UnixMilli(created).UTC().Format(time.RFC3339), "revoked": revoked != 0})
	}
	s.json(w, r, 200, map[string]any{"items": items})
}
func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 80 {
		s.jsonError(w, r, 400, "INVALID_TOKEN_NAME")
		return
	}
	raw := randomToken(32)
	id := store.ID("tok_")
	if _, err := s.Store.DB.Exec(`INSERT INTO api_tokens(id,admin_id,name,token_hash,created_at) VALUES(?,?,?,?,?)`, id, p.ID, strings.TrimSpace(in.Name), tokenHash(raw), store.Now()); err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	s.json(w, r, 201, map[string]any{"id": id, "name": strings.TrimSpace(in.Name), "token": raw})
}
func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	res, err := s.Store.DB.Exec(`UPDATE api_tokens SET revoked_at=? WHERE id=? AND admin_id=? AND revoked_at=0`, store.Now(), r.PathValue("id"), p.ID)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s.jsonError(w, r, 404, "TOKEN_NOT_FOUND")
		return
	}
	s.json(w, r, 200, map[string]bool{"revoked": true})
}
