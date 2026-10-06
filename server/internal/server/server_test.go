package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"net/netip"
)

func TestThemeLifecycle(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Mode: "lan", ListenAddr: "127.0.0.1:0", DatabasePath: filepath.Join(root, "themes.db"), DataDir: root, AllowLANHTTP: true, ThemeAccess: "anonymous", OfflinePolicy: "allow", MaxThemeBytes: 128 << 20, InstanceID: "test"}
	cfg.AdminCIDRs = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	var cookie *http.Cookie
	do := func(method, path string, body string, headers map[string]string) (*httptest.ResponseRecorder, *http.Request) {
		req := httptest.NewRequest(method, "http://test.local"+path, strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:1234"
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		app.Handler().ServeHTTP(rr, req)
		if c := rr.Result().Cookies(); len(c) > 0 {
			cookie = c[0]
		}
		return rr, req
	}
	postJSON := func(path string, v any) (int, map[string]any) {
		b, _ := json.Marshal(v)
		res, _ := do("POST", path, string(b), map[string]string{"Content-Type": "application/json"})
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.Code, out
	}
	if status, out := postJSON("/admin/v1/auth/register", map[string]any{"email": "admin@example.com", "password": "a-very-strong-password"}); status != 201 {
		t.Fatalf("register=%d out=%v", status, out)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "dist", "nora-aurora-1.0.0.noratheme"))
	if err != nil {
		t.Skip(err)
	}
	// Use multipart writer so the test exercises the same upload path as the UI.
	var body strings.Builder
	body.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"theme.noratheme\"\r\nContent-Type: application/octet-stream\r\n\r\n")
	body.Write(raw)
	body.WriteString("\r\n--boundary--\r\n")
	csrf := csrfFromSession(t, do)
	res, _ := do("POST", "/admin/v1/themes/uploads", body.String(), map[string]string{"Content-Type": "multipart/form-data; boundary=boundary", "X-CSRF-Token": csrf})
	if res.Code != 201 {
		t.Fatalf("upload=%d %s", res.Code, res.Body.String())
	}
	var themeID string
	if err := app.Store.DB.QueryRow(`SELECT id FROM theme_catalog WHERE package_id='nora-aurora'`).Scan(&themeID); err != nil {
		t.Fatal(err)
	}
	bpub, _ := json.Marshal(map[string]any{"version": "1.0.0"})
	pub, _ := do("POST", "/admin/v1/themes/"+themeID+"/publish", string(bpub), map[string]string{"Content-Type": "application/json", "X-CSRF-Token": csrf})
	if pub.Code != 200 {
		t.Fatalf("publish=%d %s", pub.Code, pub.Body.String())
	}
	listed, _ := do("GET", "/api/v1/themes", "", nil)
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), "nora-aurora") {
		t.Fatalf("list=%d %s", listed.Code, listed.Body.String())
	}
	b, _ := json.Marshal(map[string]any{"version": "1.0.0"})
	missing, _ := do("POST", "/admin/v1/themes/thm_missing/publish", string(b), map[string]string{"Content-Type": "application/json", "X-CSRF-Token": csrf})
	code := missing.Code
	if code != 409 && code != 404 {
		t.Fatalf("unexpected missing publish=%d", code)
	}
}
func csrfFromSession(t *testing.T, do func(string, string, string, map[string]string) (*httptest.ResponseRecorder, *http.Request)) string {
	t.Helper()
	res, _ := do("GET", "/admin/v1/auth/session", "", nil)
	var out struct {
		Data struct {
			CSRFToken string `json:"csrfToken"`
		} `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return out.Data.CSRFToken
}
