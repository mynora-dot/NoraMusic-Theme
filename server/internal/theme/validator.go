package theme

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

const (
	MaxFiles = 512
	MaxFile  = 32 << 20
	MaxTotal = 128 << 20
	MaxRatio = 200
)

var (
	idRE      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	versionRE = regexp.MustCompile(`^\d{1,5}\.\d{1,5}\.\d{1,5}$`)
	colorRE   = regexp.MustCompile(`^#[a-fA-F0-9]{6}([a-fA-F0-9]{2})?$`)
)

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Author        struct {
		Name string `json:"name"`
	} `json:"author"`
	Description       string   `json:"description"`
	Tags              []string `json:"tags"`
	License           string   `json:"license"`
	Preview           string   `json:"preview"`
	BrightnessSupport string   `json:"brightnessSupport"`
	Category          string   `json:"category"`
}

type Checked struct {
	Manifest      Manifest
	ManifestJSON  []byte
	ThemeJSON     []byte
	Expanded      int64
	SHA256        string
	Size          int64
	PreviewColors []uint32
}

type Error struct{ Code string }

func (e *Error) Error() string  { return e.Code }
func invalid(code string) error { return &Error{Code: code} }

func Validate(raw []byte, maxUpload int64) (Checked, error) {
	var out Checked
	if int64(len(raw)) > maxUpload {
		return out, invalid("THEME_SIZE_LIMIT")
	}
	h := sha256.Sum256(raw)
	out.SHA256 = hex.EncodeToString(h[:])
	out.Size = int64(len(raw))
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return out, invalid("INVALID_THEME_ARCHIVE")
	}
	if len(z.File) > MaxFiles {
		return out, invalid("THEME_TOO_MANY_FILES")
	}
	files := map[string][]byte{}
	seen := map[string]bool{}
	for _, f := range z.File {
		name := strings.TrimSuffix(f.Name, "/")
		if name == "" {
			continue
		}
		if !safePath(name) || seen[strings.ToLower(name)] || f.Mode()&0xF000 == 0xA000 {
			return out, invalid("UNSAFE_THEME_PATH")
		}
		seen[strings.ToLower(name)] = true
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(path.Ext(name))
		switch ext {
		case ".json", ".png", ".jpg", ".jpeg", ".webp", ".ttf", ".otf":
		default:
			return out, invalid("UNSUPPORTED_THEME_FILE")
		}
		if f.UncompressedSize64 > MaxFile || f.UncompressedSize64 > uint64(MaxRatio)*max64(f.CompressedSize64, 1) {
			return out, invalid("THEME_EXPANSION_LIMIT")
		}
		out.Expanded += int64(f.UncompressedSize64)
		if out.Expanded > MaxTotal {
			return out, invalid("THEME_EXPANSION_LIMIT")
		}
		rc, e := f.Open()
		if e != nil {
			return out, invalid("INVALID_THEME_FILE")
		}
		data, e := io.ReadAll(io.LimitReader(rc, MaxFile+1))
		_ = rc.Close()
		if e != nil || len(data) > MaxFile {
			return out, invalid("INVALID_THEME_FILE")
		}
		files[name] = data
	}
	manifest, ok := files["manifest.json"]
	if !ok {
		return out, invalid("THEME_MANIFEST_REQUIRED")
	}
	if len(manifest) > 64<<10 {
		return out, invalid("INVALID_THEME_MANIFEST")
	}
	if json.Unmarshal(manifest, &out.Manifest) != nil {
		return out, invalid("INVALID_THEME_MANIFEST")
	}
	m := out.Manifest
	if m.SchemaVersion != 2 || !idRE.MatchString(m.ID) || m.ID == "nora-light" || m.ID == "nora-dark" || !versionRE.MatchString(m.Version) || strings.TrimSpace(m.Name) == "" || len([]byte(m.Name)) > 160 || len([]byte(m.Description)) > 2000 || len([]byte(m.Author.Name)) > 160 || len(m.Tags) > 12 || len([]byte(m.Category)) > 64 {
		return out, invalid("INVALID_THEME_MANIFEST")
	}
	if m.BrightnessSupport != "" && m.BrightnessSupport != "light" && m.BrightnessSupport != "dark" && m.BrightnessSupport != "dual" {
		return out, invalid("INVALID_BRIGHTNESS_SUPPORT")
	}
	for _, tag := range m.Tags {
		if len([]byte(tag)) > 64 {
			return out, invalid("INVALID_THEME_MANIFEST")
		}
	}
	cfg, ok := files["theme.json"]
	if !ok {
		return out, invalid("THEME_CONFIG_REQUIRED")
	}
	if len(cfg) > 1<<20 {
		return out, invalid("INVALID_THEME_CONFIG")
	}
	var doc map[string]any
	if json.Unmarshal(cfg, &doc) != nil || doc == nil {
		return out, invalid("INVALID_THEME_CONFIG")
	}
	if doc["schemaVersion"] != float64(2) {
		return out, invalid("INVALID_THEME_CONFIG")
	}
	colors, ok := doc["colors"].(map[string]any)
	if !ok || len(colors) == 0 {
		return out, invalid("THEME_COLORS_REQUIRED")
	}
	if _, ok := colors["background"]; ok {
		return out, invalid("INVALID_THEME_COLOR")
	}
	for _, v := range colors {
		s, ok := v.(string)
		if !ok || !colorRE.MatchString(s) {
			return out, invalid("INVALID_THEME_COLOR")
		}
	}
	for _, key := range []string{"canvas", "accent", "textPrimary"} {
		if s, ok := colors[key].(string); ok {
			n, _ := parseColor(s)
			out.PreviewColors = append(out.PreviewColors, n)
		}
	}
	if err := walk(doc, files, ""); err != nil {
		return out, err
	}
	var md map[string]any
	_ = json.Unmarshal(manifest, &md)
	if err := walk(md, files, ""); err != nil {
		return out, err
	}
	out.ManifestJSON, out.ThemeJSON = append([]byte(nil), manifest...), append([]byte(nil), cfg...)
	return out, nil
}

func max64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
func safePath(name string) bool {
	return name != "" && len(name) <= 240 && !strings.ContainsAny(name, "\\:\x00") && !strings.HasPrefix(name, "/") && path.Clean(name) == name && !strings.HasPrefix(name, "../") && name != ".." && !strings.HasPrefix(name, "__MACOSX/") && path.Base(name) != ".DS_Store"
}
func parseColor(s string) (uint32, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		s = "ff" + s
	}
	var n uint32
	_, err := fmt.Sscanf(s, "%x", &n)
	return n, err
}
func walk(v any, files map[string][]byte, key string) error {
	switch x := v.(type) {
	case map[string]any:
		for k, item := range x {
			l := strings.ToLower(k)
			if strings.HasPrefix(l, "script") || strings.HasSuffix(l, "script") || strings.HasPrefix(l, "lua") {
				return invalid("THEME_SCRIPTS_DISABLED")
			}
			if err := walk(item, files, k); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range x {
			if err := walk(item, files, key); err != nil {
				return err
			}
		}
	case string:
		if strings.Contains(x, "://") || strings.HasPrefix(strings.ToLower(x), "file:") || strings.HasPrefix(strings.ToLower(x), "data:") {
			return invalid("EXTERNAL_THEME_RESOURCE")
		}
		if key == "path" || key == "background" || key == "backgroundImage" || key == "preview" {
			if !safePath(x) || files[x] == nil {
				return invalid("MISSING_THEME_RESOURCE")
			}
		}
	}
	return nil
}
