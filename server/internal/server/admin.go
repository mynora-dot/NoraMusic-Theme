package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminGate(w, r, false); !ok {
		return
	}
	var themes, published, drafts int
	var bytes int64
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM theme_catalog WHERE deleted_at=0`).Scan(&themes)
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM theme_versions WHERE status='published'`).Scan(&published)
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM theme_versions WHERE status='draft'`).Scan(&drafts)
	_ = s.Store.DB.QueryRow(`SELECT COALESCE(SUM(size_bytes),0) FROM theme_versions WHERE status!='deleted'`).Scan(&bytes)
	p := s.policy()
	s.json(w, r, 200, map[string]any{"themes": themes, "publishedThemes": published, "drafts": drafts, "storageBytes": bytes, "mode": s.Config.Mode, "themeAccess": p.ThemeAccess, "offlinePolicy": p.OfflinePolicy})
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminGate(w, r, false); !ok {
		return
	}
	p := s.policy()
	s.json(w, r, 200, map[string]any{"deployment": map[string]any{"mode": s.Config.Mode, "listenAddr": s.Config.ListenAddr, "allowLanHttp": s.Config.AllowLANHTTP, "allowInsecureAdmin": s.Config.AllowInsecureAdmin, "adminCidrs": s.Config.AdminCIDRs, "trustedProxyCidrs": s.Config.TrustedProxyCIDRs}, "policy": p})
}
func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Policy Policy `json:"policy"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	if (in.Policy.ThemeAccess != "anonymous" && in.Policy.ThemeAccess != "login") || (in.Policy.OfflinePolicy != "allow" && in.Policy.OfflinePolicy != "strict") || in.Policy.MaxThemeMiB < 1 || in.Policy.MaxThemeMiB > 128 {
		s.jsonError(w, r, 400, "INVALID_POLICY")
		return
	}
	if err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO settings(key,value) VALUES('policy',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, encodeJSON(in.Policy)); err != nil {
			return err
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "settings.update", "policy", in.Policy)
	}); err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	s.json(w, r, 200, in.Policy)
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminGate(w, r, false); !ok {
		return
	}
	rows, err := s.Store.DB.Query(`SELECT action,target,details_json,created_at FROM audit_events ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var action, target, details string
		var at int64
		_ = rows.Scan(&action, &target, &details, &at)
		items = append(items, map[string]any{"action": action, "target": target, "details": jsonValue(details), "createdAt": time.UnixMilli(at).UTC().Format(time.RFC3339)})
	}
	s.json(w, r, 200, map[string]any{"items": items})
}
func jsonValue(raw string) any {
	var v any
	if err := jsonUnmarshal([]byte(raw), &v); err != nil {
		return map[string]any{}
	}
	return v
}
func encodeJSON(v any) string             { b, _ := json.Marshal(v); return string(b) }
func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
