package server

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mynora/noramusic-theme-server/internal/store"
	"github.com/mynora/noramusic-theme-server/internal/theme"
)

type themeItem struct {
	ID                string   `json:"id"`
	PackageID         string   `json:"packageId"`
	Name              string   `json:"name"`
	Author            string   `json:"author"`
	Description       string   `json:"description"`
	Version           string   `json:"version"`
	Tags              []string `json:"tags"`
	BrightnessSupport string   `json:"brightnessSupport"`
	Category          string   `json:"category"`
	PreviewColors     []uint32 `json:"previewColors"`
	SHA256            string   `json:"sha256"`
	SizeBytes         int64    `json:"sizeBytes"`
	IsPackage         bool     `json:"isPackage"`
	DownloadURL       string   `json:"downloadUrl"`
}

func (s *Server) publicAllowed(w http.ResponseWriter, r *http.Request) bool {
	if s.policy().ThemeAccess != "login" {
		return true
	}
	_, ok := s.adminFromRequest(r)
	if !ok {
		s.jsonError(w, r, 401, "AUTH_REQUIRED")
	}
	return ok
}
func (s *Server) publicThemes(w http.ResponseWriter, r *http.Request) {
	if !s.publicAllowed(w, r) {
		return
	}
	rows, err := s.Store.DB.QueryContext(r.Context(), `SELECT c.id,c.package_id,c.name,c.author,c.description,c.tags_json,c.brightness_support,c.category,v.version,v.sha256,v.size_bytes,v.preview_colors_json FROM theme_catalog c JOIN theme_versions v ON v.theme_id=c.id AND v.status='published' WHERE c.deleted_at=0 ORDER BY c.updated_at DESC`)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	defer rows.Close()
	items := []themeItem{}
	for rows.Next() {
		var x themeItem
		var tags, colors string
		if err := rows.Scan(&x.ID, &x.PackageID, &x.Name, &x.Author, &x.Description, &tags, &x.BrightnessSupport, &x.Category, &x.Version, &x.SHA256, &x.SizeBytes, &colors); err != nil {
			continue
		}
		if x.BrightnessSupport == "" {
			x.BrightnessSupport = "light"
		}
		x.Tags = store.ParseJSON[[]string](tags)
		x.PreviewColors = store.ParseJSON[[]uint32](colors)
		x.IsPackage = true
		x.DownloadURL = fmt.Sprintf("api/v1/themes/%s/versions/%s/download", x.ID, x.Version)
		items = append(items, x)
	}
	s.json(w, r, 200, map[string]any{"items": items, "limit": 200})
}
func (s *Server) publicTheme(w http.ResponseWriter, r *http.Request) {
	if !s.publicAllowed(w, r) {
		return
	}
	id := r.PathValue("id")
	var x themeItem
	var tags, colors string
	err := s.Store.DB.QueryRow(`SELECT c.id,c.package_id,c.name,c.author,c.description,c.tags_json,c.brightness_support,c.category,v.version,v.sha256,v.size_bytes,v.preview_colors_json FROM theme_catalog c JOIN theme_versions v ON v.theme_id=c.id AND v.status='published' WHERE c.id=? AND c.deleted_at=0`, id).Scan(&x.ID, &x.PackageID, &x.Name, &x.Author, &x.Description, &tags, &x.BrightnessSupport, &x.Category, &x.Version, &x.SHA256, &x.SizeBytes, &colors)
	if err == sql.ErrNoRows {
		s.jsonError(w, r, 404, "THEME_NOT_FOUND")
		return
	}
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	x.Tags = store.ParseJSON[[]string](tags)
	if x.BrightnessSupport == "" {
		x.BrightnessSupport = "light"
	}
	x.PreviewColors = store.ParseJSON[[]uint32](colors)
	x.IsPackage = true
	x.DownloadURL = fmt.Sprintf("api/v1/themes/%s/versions/%s/download", x.ID, x.Version)
	s.json(w, r, 200, x)
}
func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	if !s.publicAllowed(w, r) {
		return
	}
	var key, hash string
	var size int64
	err := s.Store.DB.QueryRow(`SELECT v.storage_key,v.sha256,v.size_bytes FROM theme_versions v JOIN theme_catalog c ON c.id=v.theme_id WHERE v.theme_id=? AND v.version=? AND v.status='published' AND c.deleted_at=0`, r.PathValue("id"), r.PathValue("version")).Scan(&key, &hash, &size)
	if err == sql.ErrNoRows {
		s.jsonError(w, r, 404, "THEME_VERSION_NOT_FOUND")
		return
	}
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	if key != hash+".noratheme" {
		s.jsonError(w, r, 500, "INVALID_THEME_STORAGE")
		return
	}
	f, err := os.Open(s.Store.BlobPath(key))
	if err != nil {
		s.jsonError(w, r, 404, "THEME_FILE_NOT_FOUND")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=theme.noratheme")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "theme.noratheme", time.Time{}, f)
}
func (s *Server) validateInstalled(w http.ResponseWriter, r *http.Request) {
	if !s.publicAllowed(w, r) {
		return
	}
	var in struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		SHA256  string `json:"sha256"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	var n int
	_ = s.Store.DB.QueryRow(`SELECT COUNT(*) FROM theme_versions v JOIN theme_catalog c ON c.id=v.theme_id WHERE c.package_id=? AND v.version=? AND v.sha256=? AND v.status='published' AND c.deleted_at=0`, in.ID, in.Version, in.SHA256).Scan(&n)
	if n != 1 {
		s.jsonError(w, r, 410, "THEME_UNAVAILABLE")
		return
	}
	s.json(w, r, 200, map[string]any{"valid": true, "leaseSeconds": 300, "serverInstanceId": s.Config.InstanceID})
}
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	maxBytes := s.policy().MaxThemeMiB << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
	if err := r.ParseMultipartForm(maxBytes + 1<<20); err != nil {
		s.jsonError(w, r, 413, "THEME_SIZE_LIMIT")
		return
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File["file"]) != 1 {
		s.jsonError(w, r, 400, "ONE_THEME_FILE_REQUIRED")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header == nil {
		s.jsonError(w, r, 400, "THEME_FILE_REQUIRED")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || int64(len(raw)) > maxBytes {
		s.jsonError(w, r, 413, "THEME_SIZE_LIMIT")
		return
	}
	checked, err := theme.Validate(raw, maxBytes)
	if err != nil {
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 400, e.Code)
		} else {
			s.jsonError(w, r, 400, "INVALID_THEME_ARCHIVE")
		}
		return
	}
	key := checked.SHA256 + ".noratheme"
	tmp, err := os.CreateTemp(s.Store.BlobDir, "upload-")
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(raw); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	if _, err = os.Stat(s.Store.BlobPath(key)); os.IsNotExist(err) {
		if err = os.Rename(tmpName, s.Store.BlobPath(key)); err != nil {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
			return
		}
	}
	m := checked.Manifest
	catalogID, versionID := store.ID("thm_"), store.ID("thv_")
	tags, colors := store.JSON(m.Tags), store.JSON(checked.PreviewColors)
	err = s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		var existing string
		err := tx.QueryRowContext(r.Context(), `SELECT id FROM theme_catalog WHERE package_id=?`, m.ID).Scan(&existing)
		if err == sql.ErrNoRows {
			catalogID = store.ID("thm_")
			_, err = tx.ExecContext(r.Context(), `INSERT INTO theme_catalog(id,package_id,name,author,description,tags_json,brightness_support,category,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, catalogID, m.ID, m.Name, m.Author.Name, m.Description, tags, m.BrightnessSupport, m.Category, store.Now(), store.Now())
		} else if err == nil {
			catalogID = existing
			_, err = tx.ExecContext(r.Context(), `UPDATE theme_catalog SET name=?,author=?,description=?,tags_json=?,brightness_support=?,category=?,updated_at=?,deleted_at=0 WHERE id=?`, m.Name, m.Author.Name, m.Description, tags, m.BrightnessSupport, m.Category, store.Now(), catalogID)
		}
		if err != nil {
			return err
		}
		var n int
		_ = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM theme_versions WHERE theme_id=? AND version=?`, catalogID, m.Version).Scan(&n)
		if n > 0 {
			return &theme.Error{Code: "THEME_VERSION_EXISTS"}
		}
		_, err = tx.ExecContext(r.Context(), `INSERT INTO theme_versions(id,theme_id,version,sha256,size_bytes,expanded_bytes,storage_key,manifest_json,theme_json,preview_colors_json,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, versionID, catalogID, m.Version, checked.SHA256, checked.Size, checked.Expanded, key, string(checked.ManifestJSON), string(checked.ThemeJSON), colors, "draft", store.Now())
		if err != nil {
			return err
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.upload", catalogID+"@"+m.Version, map[string]any{"sha256": checked.SHA256})
	})
	if err != nil {
		s.Store.RemoveBlobIfUnreferenced(r.Context(), key)
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 409, e.Code)
		} else {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
		}
		return
	}
	s.json(w, r, 201, map[string]any{"id": catalogID, "versionId": versionID, "version": m.Version, "sha256": checked.SHA256, "status": "draft"})
}
func (s *Server) adminThemes(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, false)
	_ = p
	if !ok {
		return
	}
	rows, err := s.Store.DB.Query(`SELECT c.id,c.package_id,c.name,c.author,c.deleted_at,(SELECT v.version FROM theme_versions v WHERE v.theme_id=c.id ORDER BY v.created_at DESC LIMIT 1),(SELECT v.status FROM theme_versions v WHERE v.theme_id=c.id ORDER BY v.created_at DESC LIMIT 1),(SELECT v.version FROM theme_versions v WHERE v.theme_id=c.id AND v.status='published' LIMIT 1),COUNT(v.id),c.updated_at FROM theme_catalog c LEFT JOIN theme_versions v ON v.theme_id=c.id GROUP BY c.id ORDER BY c.updated_at DESC`)
	if err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, pkg, name, author, latest, status, published string
		var deleted int
		var count int
		var updated int64
		_ = rows.Scan(&id, &pkg, &name, &author, &deleted, &latest, &status, &published, &count, &updated)
		items = append(items, map[string]any{"id": id, "package_id": pkg, "name": name, "author": author, "latest_version": latest, "latest_status": status, "published_version": published, "version_count": count, "deleted": deleted != 0, "updated_at": time.UnixMilli(updated).UTC().Format(time.RFC3339)})
	}
	s.json(w, r, 200, map[string]any{"items": items})
}
func (s *Server) publish(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Version string `json:"version"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		var status string
		if tx.QueryRow(`SELECT status FROM theme_versions WHERE theme_id=? AND version=?`, id, in.Version).Scan(&status) != nil {
			return &theme.Error{Code: "THEME_VERSION_NOT_FOUND"}
		}
		if status != "draft" {
			return &theme.Error{Code: "THEME_VERSION_NOT_DRAFT"}
		}
		_, _ = tx.Exec(`UPDATE theme_versions SET status='withdrawn',withdrawn_at=? WHERE theme_id=? AND status='published'`, store.Now(), id)
		_, err := tx.Exec(`UPDATE theme_versions SET status='published',published_at=? WHERE theme_id=? AND version=?`, store.Now(), id, in.Version)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE theme_catalog SET updated_at=?,deleted_at=0 WHERE id=?`, store.Now(), id)
		if err != nil {
			return err
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.publish", id+"@"+in.Version, nil)
	})
	if err != nil {
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 409, e.Code)
		} else {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
		}
		return
	}
	s.json(w, r, 200, map[string]bool{"published": true})
}
func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Version string `json:"version"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		res, e := tx.Exec(`UPDATE theme_versions SET status='withdrawn',withdrawn_at=? WHERE theme_id=? AND version=? AND status='published'`, store.Now(), id, in.Version)
		if e != nil {
			return e
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return &theme.Error{Code: "THEME_VERSION_NOT_FOUND"}
		}
		_, e = tx.Exec(`UPDATE theme_catalog SET updated_at=? WHERE id=?`, store.Now(), id)
		if e != nil {
			return e
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.withdraw", id+"@"+in.Version, nil)
	})
	if err != nil {
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 404, e.Code)
		} else {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
		}
		return
	}
	s.json(w, r, 200, map[string]bool{"withdrawn": true})
}
func (s *Server) deleteTheme(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("id")
	err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		res, e := tx.Exec(`UPDATE theme_catalog SET deleted_at=?,updated_at=? WHERE id=? AND deleted_at=0`, store.Now(), store.Now(), id)
		if e != nil {
			return e
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return &theme.Error{Code: "THEME_NOT_FOUND"}
		}
		_, e = tx.Exec(`UPDATE theme_versions SET status='deleted',deleted_at=? WHERE theme_id=? AND status!='deleted'`, store.Now(), id)
		if e != nil {
			return e
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.delete", id, nil)
	})
	if err != nil {
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 404, e.Code)
		} else {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
		}
		return
	}
	s.json(w, r, 200, map[string]bool{"deleted": true})
}
func (s *Server) restoreTheme(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("id")
	err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		res, e := tx.Exec(`UPDATE theme_catalog SET deleted_at=0,updated_at=? WHERE id=? AND deleted_at!=0`, store.Now(), id)
		if e != nil {
			return e
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return &theme.Error{Code: "THEME_NOT_FOUND"}
		}
		_, e = tx.Exec(`UPDATE theme_versions SET status='withdrawn',deleted_at=0 WHERE theme_id=? AND status='deleted'`, id)
		if e != nil {
			return e
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.restore", id, nil)
	})
	if err != nil {
		if e, ok := err.(*theme.Error); ok {
			s.jsonError(w, r, 404, e.Code)
		} else {
			s.jsonError(w, r, 500, "INTERNAL_ERROR")
		}
		return
	}
	s.json(w, r, 200, map[string]bool{"restored": true})
}
func (s *Server) purgeTheme(w http.ResponseWriter, r *http.Request) {
	p, ok := s.adminGate(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("id")
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !s.decodeJSON(w, r, &in) {
		return
	}
	var pkg string
	if s.Store.DB.QueryRow(`SELECT package_id FROM theme_catalog WHERE id=?`, id).Scan(&pkg) != nil || in.Confirm != pkg {
		s.jsonError(w, r, 400, "PURGE_CONFIRMATION_REQUIRED")
		return
	}
	var keys []string
	func() {
		rows, _ := s.Store.DB.Query(`SELECT storage_key FROM theme_versions WHERE theme_id=?`, id)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var k string
				_ = rows.Scan(&k)
				keys = append(keys, k)
			}
		}
	}()
	if err := s.Store.Transaction(r.Context(), func(tx *sql.Tx) error {
		if _, e := tx.Exec(`DELETE FROM theme_catalog WHERE id=?`, id); e != nil {
			return e
		}
		return s.Store.Audit(r.Context(), tx, p.ID, "theme.purge", id, nil)
	}); err != nil {
		s.jsonError(w, r, 500, "INTERNAL_ERROR")
		return
	}
	for _, k := range keys {
		s.Store.RemoveBlobIfUnreferenced(r.Context(), k)
	}
	s.json(w, r, 200, map[string]bool{"purged": true})
}
