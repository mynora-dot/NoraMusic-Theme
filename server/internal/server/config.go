package server

import (
	"crypto/sha256"
	"errors"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Mode               string
	ListenAddr         string
	DatabasePath       string
	DataDir            string
	PublicBaseURL      string
	MaxThemeBytes      int64
	AdminCIDRs         []netip.Prefix
	TrustedProxyCIDRs  []netip.Prefix
	AllowLANHTTP       bool
	AllowInsecureAdmin bool
	ThemeAccess        string
	OfflinePolicy      string
	InstanceID         string
}

func LoadConfig() (Config, error) {
	mode := env("NORA_DEPLOY_MODE", "public")
	if mode != "public" && mode != "lan" {
		return Config{}, errors.New("NORA_DEPLOY_MODE must be public or lan")
	}
	allowHTTP := env("ALLOW_LAN_HTTP", "false") == "true"
	if allowHTTP && mode != "lan" {
		return Config{}, errors.New("ALLOW_LAN_HTTP requires lan mode")
	}
	allowInsecure := env("ALLOW_INSECURE_ADMIN", "false") == "true"
	if allowInsecure && mode != "lan" {
		return Config{}, errors.New("ALLOW_INSECURE_ADMIN requires lan mode")
	}
	maxMiB, err := strconv.ParseInt(env("MAX_THEME_MIB", "32"), 10, 64)
	if err != nil || maxMiB < 1 || maxMiB > 128 {
		return Config{}, errors.New("MAX_THEME_MIB must be 1..128")
	}
	admin, err := parseCIDRs(env("ADMIN_ALLOWED_CIDRS", "127.0.0.1/32,::1/128"))
	if err != nil {
		return Config{}, err
	}
	proxy, err := parseCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		return Config{}, err
	}
	base := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if mode == "public" {
		if base == "" {
			return Config{}, errors.New("PUBLIC_BASE_URL is required in public mode")
		}
		u, e := url.Parse(base)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return Config{}, errors.New("PUBLIC_BASE_URL must be an HTTPS URL without credentials, query or fragment")
		}
	} else if base == "" {
		base = "http://127.0.0.1:8090"
	}
	access := env("THEME_ACCESS", "anonymous")
	if access != "anonymous" && access != "login" {
		return Config{}, errors.New("THEME_ACCESS must be anonymous or login")
	}
	offline := env("OFFLINE_POLICY", "allow")
	if offline != "allow" && offline != "strict" {
		return Config{}, errors.New("OFFLINE_POLICY must be allow or strict")
	}
	db := env("DATABASE_PATH", "")
	data := env("DATA_DIR", "./data")
	if db == "" {
		db = filepath.Join(data, "themes.db")
	}
	return Config{Mode: mode, ListenAddr: env("LISTEN_ADDR", "127.0.0.1:8090"), DatabasePath: db, DataDir: data, PublicBaseURL: base, MaxThemeBytes: maxMiB << 20, AdminCIDRs: admin, TrustedProxyCIDRs: proxy, AllowLANHTTP: allowHTTP, AllowInsecureAdmin: allowInsecure, ThemeAccess: access, OfflinePolicy: offline, InstanceID: env("SERVER_INSTANCE_ID", "noramusic-theme-store")}, nil
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func parseCIDRs(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range strings.Split(raw, ",") {
		if strings.TrimSpace(item) == "" {
			continue
		}
		p, err := netip.ParsePrefix(strings.TrimSpace(item))
		if err != nil {
			return nil, errors.New("invalid CIDR configuration")
		}
		out = append(out, p.Masked())
	}
	return out, nil
}
func digest(v string) [32]byte { return sha256.Sum256([]byte(v)) }
