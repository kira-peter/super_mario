package editor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"super-mario/pkg/catalog"
	"super-mario/pkg/level"
	"super-mario/pkg/version"
)

// Server is the editor's HTTP side.
//
// It is reachable only from this machine, answers only requests that come from
// its own page, and only ever writes level files into the two level folders,
// under names it has checked itself. Those three rules are what make it safe
// to run a server that writes into the repository (design decision 1).
type Server struct {
	// Catalog is where levels are read and written.
	Catalog *catalog.Catalog
	// Open is the level to show first, as a catalog key, if any.
	Open     string
	draftDir string
	saveMu   sync.Mutex

	// addr is what the server listens on, for checking Host and Origin.
	addr string
}

// ValidID reports whether id is a usable name for a saved level, and if not,
// why. The reason is a code the page translates.
func ValidID(id string) error {
	if !catalog.ValidName(id) {
		return errors.New("bad-id-chars")
	}
	if !catalog.ValidID(id) {
		return errors.New("bad-id-form")
	}
	return nil
}

// Handler returns the server's routes. addr is the host:port it serves on.
func (s *Server) Handler(addr string) http.Handler {
	s.addr = addr
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/levels", s.list)
	mux.HandleFunc("GET /api/levels/{source}/{id}", s.read)
	mux.HandleFunc("PUT /api/levels/{source}/{id}", s.write)
	mux.HandleFunc("GET /api/info", s.info)
	mux.HandleFunc("GET /api/drafts", s.readDraft)
	mux.HandleFunc("PUT /api/drafts/{id}", s.writeDraft)
	mux.HandleFunc("DELETE /api/drafts/{id}", s.deleteDraft)
	mux.Handle("GET /", http.FileServerFS(Web()))
	return s.guard(mux)
}

// guard turns away anything that is not this editor's own page talking to it.
//
// Host must name the loopback address: a web page elsewhere can point a
// hostname at 127.0.0.1 (DNS rebinding), but its requests still carry that
// hostname. Origin, when present, must be this server: a page on another
// origin can send a request here even without reading the answer, and must
// not be able to save a level that way.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.localHost(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && !s.ownOrigin(o) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		// A write must say where it comes from. Browsers always send
		// Origin on a PUT, so a write without one is not from a page.
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Origin") == "" {
			http.Error(w, "missing origin", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) localHost(host string) bool {
	h, port, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}
	_, ownPort, _ := net.SplitHostPort(s.addr)
	if port != ownPort {
		return false
	}
	return h == "127.0.0.1" || h == "localhost" || h == "[::1]" || h == "::1"
}

func (s *Server) ownOrigin(origin string) bool {
	rest, ok := strings.CutPrefix(origin, "http://")
	return ok && s.localHost(rest)
}

// --- routes ---------------------------------------------------------------

func (s *Server) info(w http.ResponseWriter, _ *http.Request) {
	c := s.Catalog
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inRepo":     c.InRepo(),
		"builtinDir": c.BuiltinDir,
		"localDir":   c.LocalDir,
		"newSource":  c.NewLevelSource(),
		"writable": map[catalog.Source]bool{
			catalog.Builtin: c.Writable(catalog.Builtin),
			catalog.Local:   c.Writable(catalog.Local),
		},
		"open": s.Open,
		"wasm": HasWasm(),
		// The version the editor belongs to, so a problem report can name it.
		"version": version.String(),
	})
}

func (s *Server) list(w http.ResponseWriter, _ *http.Request) {
	out := s.Catalog.List()
	if out == nil {
		out = []catalog.Entry{}
	}
	writeJSON(w, http.StatusOK, out)
}

func source(r *http.Request) (catalog.Source, bool) {
	switch catalog.Source(r.PathValue("source")) {
	case catalog.Builtin:
		return catalog.Builtin, true
	case catalog.Local:
		return catalog.Local, true
	}
	return "", false
}

func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	src, ok := source(r)
	id := r.PathValue("id")
	if !ok || !catalog.ValidName(id) {
		http.Error(w, "bad level", http.StatusBadRequest)
		return
	}
	data, err := s.Catalog.Read(src, id)
	if err != nil {
		http.Error(w, "no such level", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("ETag", levelRevision(data))
	w.Write(data)
}

func levelRevision(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// SaveRequest is what the editor sends to save.
type SaveRequest struct {
	// Level is the level file's text.
	Level string `json:"level"`
	// Overwrite confirms replacing a level that is not the one opened. The
	// editor asks the designer first and only then sets it.
	Overwrite bool `json:"overwrite"`
	// OpenedAs is the catalog key the level was opened from, if any.
	OpenedAs string `json:"openedAs"`
	// Revision identifies the exact version read or confirmed for replacement.
	Revision string `json:"revision"`
}

// saveError is a refusal the page can translate: a code, and details for the
// message.
func saveError(w http.ResponseWriter, status int, code string, extra map[string]string) {
	body := map[string]string{"code": code}
	for k, v := range extra {
		body[k] = v
	}
	writeJSON(w, status, body)
}

func (s *Server) write(w http.ResponseWriter, r *http.Request) {
	src, ok := source(r)
	if !ok {
		saveError(w, http.StatusBadRequest, "bad-source", nil)
		return
	}
	id := r.PathValue("id")
	if err := ValidID(id); err != nil {
		saveError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if !s.Catalog.Writable(src) {
		saveError(w, http.StatusConflict, "read-only", nil)
		return
	}
	var req SaveRequest
	err := readJSON(w, r, &req)
	if err != nil {
		saveError(w, http.StatusBadRequest, "bad-request", map[string]string{"detail": err.Error()})
		return
	}

	// The server checks again whatever the page says: the file must load in
	// the game.
	formatted, err := level.Format([]byte(req.Level))
	if err != nil {
		saveError(w, http.StatusUnprocessableEntity, "invalid-level", map[string]string{"detail": err.Error()})
		return
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	key := catalog.Key(src, id)
	current, err := s.Catalog.Read(src, id)
	switch {
	case err == nil:
		revision := levelRevision(current)
		if req.OpenedAs != key && !req.Overwrite {
			saveError(w, http.StatusConflict, "exists", map[string]string{"id": id, "source": string(src), "revision": revision})
			return
		}
		if revision != req.Revision {
			saveError(w, http.StatusConflict, "level-changed", nil)
			return
		}
	case !errors.Is(err, catalog.ErrNotFound):
		saveError(w, http.StatusInternalServerError, "write-failed", map[string]string{"detail": err.Error()})
		return
	case req.OpenedAs == key || req.Overwrite:
		saveError(w, http.StatusConflict, "level-changed", nil)
		return
	}
	path, err := s.Catalog.Write(src, id, []byte(req.Level))
	if err != nil {
		saveError(w, http.StatusInternalServerError, "write-failed", map[string]string{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"id": id, "source": string(src), "key": catalog.Key(src, id),
		"path": path, "text": string(formatted), "revision": levelRevision(formatted),
	})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
