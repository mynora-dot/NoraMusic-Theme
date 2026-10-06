package server

import "testing"

func TestLoadConfigSecurityModes(t *testing.T) {
	t.Setenv("NORA_DEPLOY_MODE", "public")
	t.Setenv("PUBLIC_BASE_URL", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("public mode must require PUBLIC_BASE_URL")
	}
	t.Setenv("PUBLIC_BASE_URL", "https://themes.example.com")
	t.Setenv("ALLOW_INSECURE_ADMIN", "true")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("public mode must reject insecure admin")
	}
	t.Setenv("NORA_DEPLOY_MODE", "lan")
	t.Setenv("ALLOW_INSECURE_ADMIN", "false")
	t.Setenv("ALLOW_LAN_HTTP", "true")
	t.Setenv("PUBLIC_BASE_URL", "")
	cfg, err := LoadConfig()
	if err != nil || cfg.Mode != "lan" || !cfg.AllowLANHTTP {
		t.Fatalf("lan config: %#v %v", cfg, err)
	}
}
