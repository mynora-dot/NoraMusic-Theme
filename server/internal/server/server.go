package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/mynora/noramusic-theme-server/internal/store"
)

type Server struct {
	Config       Config
	Store        *store.Store
	authMu       sync.Mutex
	authAttempts map[string][]time.Time
}

type Policy struct {
	ThemeAccess   string `json:"themeAccess"`
	OfflinePolicy string `json:"offlinePolicy"`
	MaxThemeMiB   int64  `json:"maxThemeMiB"`
}

func (s *Server) policy() Policy {
	p := Policy{ThemeAccess: s.Config.ThemeAccess, OfflinePolicy: s.Config.OfflinePolicy, MaxThemeMiB: s.Config.MaxThemeBytes >> 20}
	var raw string
	if s.Store != nil && s.Store.DB.QueryRow(`SELECT value FROM settings WHERE key='policy'`).Scan(&raw) == nil {
		var saved Policy
		if json.Unmarshal([]byte(raw), &saved) == nil {
			if saved.ThemeAccess != "" {
				p.ThemeAccess = saved.ThemeAccess
			}
			if saved.OfflinePolicy != "" {
				p.OfflinePolicy = saved.OfflinePolicy
			}
			if saved.MaxThemeMiB > 0 {
				p.MaxThemeMiB = saved.MaxThemeMiB
			}
		}
	}
	return p
}

func New(cfg Config) (*Server, error) {
	st, err := store.Open(cfg.DatabasePath, cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Server{Config: cfg, Store: st, authAttempts: make(map[string][]time.Time)}, nil
}
func (s *Server) allowAuth(r *http.Request, action string) bool {
	key := action + ":" + s.requestIP(r).String()
	now := time.Now()
	s.authMu.Lock()
	defer s.authMu.Unlock()
	list := s.authAttempts[key]
	kept := list[:0]
	for _, at := range list {
		if now.Sub(at) < time.Hour {
			kept = append(kept, at)
		}
	}
	if len(kept) >= 30 {
		s.authAttempts[key] = kept
		return false
	}
	s.authAttempts[key] = append(kept, now)
	return true
}
func (s *Server) Close() error { return s.Store.Close() }
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/health", s.health)
	m.HandleFunc("GET /api/v1/capabilities", s.capabilities)
	m.HandleFunc("GET /api/v1/themes", s.publicThemes)
	m.HandleFunc("GET /api/v1/themes/{id}", s.publicTheme)
	m.HandleFunc("GET /api/v1/themes/{id}/versions/{version}/download", s.download)
	m.HandleFunc("POST /api/v1/themes/validate-installed", s.validateInstalled)
	m.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/admin/", http.StatusFound) })
	m.HandleFunc("GET /admin/", s.adminPage)
	m.HandleFunc("GET /admin/v1/auth/bootstrap", s.bootstrap)
	m.HandleFunc("POST /admin/v1/auth/register", s.register)
	m.HandleFunc("POST /admin/v1/auth/login", s.login)
	m.HandleFunc("POST /admin/v1/auth/logout", s.logout)
	m.HandleFunc("GET /admin/v1/auth/session", s.session)
	m.HandleFunc("POST /admin/v1/auth/mfa/setup", s.setupMFA)
	m.HandleFunc("POST /admin/v1/auth/verify-mfa", s.verifyMFA)
	m.HandleFunc("GET /admin/v1/auth/tokens", s.listTokens)
	m.HandleFunc("POST /admin/v1/auth/tokens", s.createToken)
	m.HandleFunc("DELETE /admin/v1/auth/tokens/{id}", s.revokeToken)
	m.HandleFunc("GET /admin/v1/dashboard", s.dashboard)
	m.HandleFunc("GET /admin/v1/themes", s.adminThemes)
	m.HandleFunc("POST /admin/v1/themes/uploads", s.upload)
	m.HandleFunc("POST /admin/v1/themes/{id}/publish", s.publish)
	m.HandleFunc("POST /admin/v1/themes/{id}/withdraw", s.withdraw)
	m.HandleFunc("DELETE /admin/v1/themes/{id}", s.deleteTheme)
	m.HandleFunc("POST /admin/v1/themes/{id}/restore", s.restoreTheme)
	m.HandleFunc("POST /admin/v1/themes/{id}/purge", s.purgeTheme)
	m.HandleFunc("GET /admin/v1/settings", s.settings)
	m.HandleFunc("PUT /admin/v1/settings", s.updateSettings)
	m.HandleFunc("GET /admin/v1/audit-events", s.audit)
	return s.middleware(m)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := randomToken(8)
		w.Header().Set("X-Request-ID", rid)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/admin") {
			// The admin UI is a single embedded document, so its trusted inline
			// script/style need explicit CSP permission; all dynamic text is escaped.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
			w.Header().Set("Cache-Control", "no-store")
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("requestId=%s method=%s path=%s durationMs=%d\n", rid, r.Method, r.URL.Path, time.Since(start).Milliseconds())
	})
}
func (s *Server) json(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "requestId": w.Header().Get("X-Request-ID"), "serverTime": time.Now().UTC().Format(time.RFC3339)})
}
func (s *Server) jsonError(w http.ResponseWriter, r *http.Request, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "message": code, "retryable": status == 429 || status >= 500, "requestId": w.Header().Get("X-Request-ID")}})
}
func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil || len(raw) > 1<<20 {
		s.jsonError(w, r, 413, "REQUEST_TOO_LARGE")
		return false
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if dec.Decode(dst) != nil {
		s.jsonError(w, r, 400, "INVALID_JSON")
		return false
	}
	return true
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if s.Store.DB.PingContext(r.Context()) != nil {
		s.jsonError(w, r, 503, "DATABASE_UNAVAILABLE")
		return
	}
	s.json(w, r, 200, map[string]string{"status": "ok"})
}
func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	p := s.policy()
	s.json(w, r, 200, map[string]any{"service": "noramusic-theme-store", "apiVersion": "v1", "serverInstanceId": s.Config.InstanceID, "themes": map[string]any{"enabled": true, "access": p.ThemeAccess, "offlinePolicy": p.OfflinePolicy, "scriptsAllowed": false, "schemaVersions": []int{2}, "maxBytes": p.MaxThemeMiB << 20}})
}
func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	if !s.authNetwork(w, r) {
		return
	}
	p, ok := s.requireAdmin(w, r, false)
	if !ok {
		return
	}
	var mfa int
	_ = s.Store.DB.QueryRow(`SELECT totp_enabled FROM admins WHERE id=?`, p.ID).Scan(&mfa)
	s.json(w, r, 200, map[string]any{"email": p.Email, "csrfToken": p.CSRF, "mfaEnabled": mfa == 1})
}
func (s *Server) adminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(adminHTML))
}
func (s *Server) requestIP(r *http.Request) netip.Addr {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip, _ := netip.ParseAddr(host)
	for _, p := range s.Config.TrustedProxyCIDRs {
		if p.Contains(ip) {
			if forwarded, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-IP"))); err == nil {
				return forwarded.Unmap()
			}
		}
	}
	return ip.Unmap()
}
func (s *Server) allowedAdminNetwork(r *http.Request) bool {
	ip := s.requestIP(r)
	for _, p := range s.Config.AdminCIDRs {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
func (s *Server) secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip, _ := netip.ParseAddr(host)
	for _, p := range s.Config.TrustedProxyCIDRs {
		if p.Contains(ip) {
			return r.Header.Get("X-Forwarded-Proto") == "https"
		}
	}
	return false
}
func (s *Server) adminGate(w http.ResponseWriter, r *http.Request, write bool) (principal, bool) {
	if !s.allowedAdminNetwork(r) {
		s.jsonError(w, r, 403, "ADMIN_NETWORK_DENIED")
		return principal{}, false
	}
	if (s.Config.Mode == "public" || (s.Config.Mode == "lan" && !s.Config.AllowLANHTTP)) && !s.secureRequest(r) {
		s.jsonError(w, r, 403, "HTTPS_REQUIRED")
		return principal{}, false
	}
	if write && r.Header.Get("Origin") != "" {
		scheme := "http"
		if s.secureRequest(r) {
			scheme = "https"
		}
		expected := scheme + "://" + r.Host
		configured := strings.TrimRight(s.Config.PublicBaseURL, "/")
		if r.Header.Get("Origin") != expected && (configured == "" || r.Header.Get("Origin") != configured) {
			s.jsonError(w, r, 403, "ORIGIN_DENIED")
			return principal{}, false
		}
	}
	p, ok := s.requireAdmin(w, r, write)
	if !ok {
		return principal{}, false
	}
	if s.Config.Mode == "public" && !strings.HasSuffix(r.URL.Path, "/auth/session") && !strings.HasSuffix(r.URL.Path, "/auth/mfa/setup") && !strings.HasSuffix(r.URL.Path, "/auth/verify-mfa") && !strings.HasSuffix(r.URL.Path, "/auth/logout") {
		var enabled int
		if s.Store.DB.QueryRow(`SELECT totp_enabled FROM admins WHERE id=?`, p.ID).Scan(&enabled) != nil || enabled != 1 {
			s.jsonError(w, r, http.StatusForbidden, "MFA_REQUIRED")
			return principal{}, false
		}
	}
	return p, true
}
func (s *Server) hashBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
