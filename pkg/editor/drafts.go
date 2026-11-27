package editor

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"super-mario/pkg/catalog"
)

type draft struct {
	Token    string         `json:"token"`
	ID       string         `json:"id"`
	Source   catalog.Source `json:"source"`
	OpenedAs string         `json:"openedAs"`
	Revision string         `json:"revision"`
	Level    string         `json:"level"`
	At       int64          `json:"at"`
}

var draftToken = regexp.MustCompile(`^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$`)

// defaultDraftDir keeps drafts for different catalogs apart, even when their
// editors use the same browser or a different loopback port.
func defaultDraftDir(c *catalog.Catalog) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	dirs := []string{c.BuiltinDir, c.LocalDir}
	for i, path := range dirs {
		if path != "" {
			if abs, err := filepath.Abs(path); err == nil {
				dirs[i] = abs
			}
		}
	}
	identity := []byte(strings.Join(dirs, "\x00"))
	return filepath.Join(dir, "super-mario", "drafts", strings.Trim(levelRevision(identity), `"`))
}

func (s *Server) readDraft(w http.ResponseWriter, r *http.Request) {
	if s.draftDir == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	entries, err := os.ReadDir(s.draftDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	type candidate struct {
		name string
		at   time.Time
	}
	var files []candidate
	for _, entry := range entries {
		if token, ok := strings.CutSuffix(entry.Name(), ".json"); !ok || !draftToken.MatchString(token) {
			continue
		}
		if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
			files = append(files, candidate{entry.Name(), info.ModTime()})
		}
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].at.After(files[j].at) })
	key := r.URL.Query().Get("key")
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(s.draftDir, file.name))
		var d draft
		if err == nil && json.Unmarshal(data, &d) == nil && d.Level != "" && d.Token+".json" == file.name &&
			(key == "" || d.OpenedAs == key) {
			writeJSON(w, http.StatusOK, d)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeDraft(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("id")
	var d draft
	if !draftToken.MatchString(token) || readJSON(w, r, &d) != nil || d.Level == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if s.draftDir == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	d.Token = token
	if d.At <= 0 {
		d.At = time.Now().UnixMilli()
	}
	data, err := json.Marshal(d)
	if err == nil {
		err = os.MkdirAll(s.draftDir, 0o700)
	}
	if err == nil {
		err = catalog.WriteAtomic(filepath.Join(s.draftDir, token+".json"), data)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteDraft(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("id")
	if !draftToken.MatchString(token) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if s.draftDir == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if err := os.Remove(filepath.Join(s.draftDir, token+".json")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
