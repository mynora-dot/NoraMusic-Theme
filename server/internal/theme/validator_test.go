package theme

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestValidateRepositoryThemes(t *testing.T) {
	for _, name := range []string{"nora-aurora", "nora-cinematic", "starter-theme"} {
		path := "../../../dist/" + name + "-1.0.0.noratheme"
		if name == "starter-theme" {
			path = "../../../dist/my-first-theme-1.0.0.noratheme"
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("fixture %s unavailable: %v", path, err)
		}
		if _, err := Validate(raw, 128<<20); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestValidateRejectsScriptsExternalAndReservedID(t *testing.T) {
	base := map[string]any{"schemaVersion": 2, "id": "safe-theme", "name": "Safe", "version": "1.0.0"}
	themeDoc := map[string]any{"schemaVersion": 2, "colors": map[string]any{"canvas": "#ffffff"}}
	makeRaw := func(manifest map[string]any, doc map[string]any) []byte {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for name, value := range map[string]any{"manifest.json": manifest, "theme.json": doc} {
			w, _ := z.Create(name)
			raw, _ := json.Marshal(value)
			_, _ = w.Write(raw)
		}
		_ = z.Close()
		return b.Bytes()
	}
	if _, err := Validate(makeRaw(base, themeDoc), 1<<20); err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}
	bad := clone(base)
	bad["script"] = "x"
	if _, err := Validate(makeRaw(bad, themeDoc), 1<<20); code(err) != "THEME_SCRIPTS_DISABLED" {
		t.Fatalf("script code=%s", code(err))
	}
	bad = clone(base)
	bad["id"] = "nora-light"
	if _, err := Validate(makeRaw(bad, themeDoc), 1<<20); code(err) != "INVALID_THEME_MANIFEST" {
		t.Fatalf("reserved code=%s", code(err))
	}
	badDoc := map[string]any{"schemaVersion": 2, "colors": map[string]any{"canvas": "#ffffff"}, "image": "https://example.com/a"}
	if _, err := Validate(makeRaw(base, badDoc), 1<<20); code(err) != "EXTERNAL_THEME_RESOURCE" {
		t.Fatalf("url code=%s", code(err))
	}
}

func clone(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func code(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ""
}
