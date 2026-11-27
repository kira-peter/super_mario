package editor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"super-mario/pkg/catalog"
)

const addr = "127.0.0.1:41234"
const origin = "http://127.0.0.1:41234"

func goodLevel(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "level", "data", "1-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// repoServer is the editor as it runs inside the repository.
func repoServer(t *testing.T) (*Server, http.Handler, string, string) {
	t.Helper()
	b, l := t.TempDir(), filepath.Join(t.TempDir(), "levels")
	os.WriteFile(filepath.Join(b, "1-1.json"), []byte(goodLevel(t)), 0o644)
	s := &Server{Catalog: &catalog.Catalog{BuiltinDir: b, LocalDir: l}}
	return s, s.Handler(addr), b, l
}

// shippedServer is the editor on a computer with only the program.
func shippedServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	l := filepath.Join(t.TempDir(), "levels")
	s := &Server{Catalog: &catalog.Catalog{LocalDir: l, Embedded: fstest.MapFS{"1-1.json": {Data: []byte(goodLevel(t))}}}}
	return s.Handler(addr), l
}

func do(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+addr+path, strings.NewReader(body))
	r.Host = addr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func save(h http.Handler, src, id string, req SaveRequest) *httptest.ResponseRecorder {
	b, _ := json.Marshal(req)
	return do(h, "PUT", "/api/levels/"+src+"/"+id, string(b), map[string]string{"Origin": origin})
}

func code(w *httptest.ResponseRecorder) string {
	return field(w, "code")
}

func field(w *httptest.ResponseRecorder, name string) string {
	var m map[string]string
	json.Unmarshal(w.Body.Bytes(), &m)
	return m[name]
}

func TestSaveWritesTheFormattedLevelAndSaysWhere(t *testing.T) {
	_, h, builtin, _ := repoServer(t)
	w := save(h, "builtin", "c-tom-sky-run", SaveRequest{Level: goodLevel(t)})
	if w.Code != http.StatusOK {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	path := filepath.Join(builtin, "c-tom-sky-run.json")
	got, err := os.ReadFile(path)
	if err != nil || string(got) != goodLevel(t) {
		t.Fatalf("file: %v", err)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["path"] != path || resp["key"] != "c-tom-sky-run" {
		t.Errorf("response %v", resp)
	}
}

func TestSaveGoesBackToTheLocalFolder(t *testing.T) {
	_, h, builtin, local := repoServer(t)
	w := save(h, "local", "c-tom-sky", SaveRequest{Level: goodLevel(t)})
	if w.Code != http.StatusOK {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(local, "c-tom-sky.json")); err != nil {
		t.Error("not in the local folder")
	}
	if _, err := os.Stat(filepath.Join(builtin, "c-tom-sky.json")); err == nil {
		t.Error("a local save went to the built-in folder")
	}
}

// TestShippedProgramSavesLocallyNeverDownloads is the bug a user hit: outside
// the repository saving used to turn into a browser download.
func TestShippedProgramSavesLocallyNeverDownloads(t *testing.T) {
	h, local := shippedServer(t)
	var info map[string]interface{}
	json.Unmarshal(do(h, "GET", "/api/info", "", nil).Body.Bytes(), &info)
	if info["newSource"] != "local" {
		t.Errorf("new levels go to %v, want local", info["newSource"])
	}
	if w := save(h, "local", "c-me-level", SaveRequest{Level: goodLevel(t)}); w.Code != http.StatusOK {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(local, "c-me-level.json")); err != nil {
		t.Error("the level was not written beside the program")
	}
	// The compiled-in levels cannot be written.
	if w := save(h, "builtin", "1-1", SaveRequest{Level: goodLevel(t), OpenedAs: "1-1"}); w.Code != http.StatusConflict || code(w) != "read-only" {
		t.Errorf("write to compiled-in: %d %s", w.Code, w.Body)
	}
}

func TestSaveRejectsBadIDs(t *testing.T) {
	_, h, builtin, local := repoServer(t)
	for _, id := range []string{"..%2f..%2fevil", "My%20Level", "UPPER", "c-", "tom-level", "a.json", "%2e%2e"} {
		if w := save(h, "local", id, SaveRequest{Level: goodLevel(t)}); w.Code == http.StatusOK {
			t.Errorf("id %q was accepted", id)
		}
	}
	if w := save(h, "elsewhere", "c-a-b", SaveRequest{Level: goodLevel(t)}); w.Code == http.StatusOK {
		t.Error("an unknown source was accepted")
	}
	entries, _ := os.ReadDir(builtin)
	if len(entries) != 1 {
		t.Errorf("files appeared in the built-in folder: %v", entries)
	}
	if _, err := os.Stat(local); err == nil {
		t.Error("the local folder was created by a refused save")
	}
}

func TestValidIDGivesACode(t *testing.T) {
	if err := ValidID("My Level!"); err == nil || err.Error() != "bad-id-chars" {
		t.Errorf("err = %v", err)
	}
	if err := ValidID("tom-level"); err == nil || err.Error() != "bad-id-form" {
		t.Errorf("err = %v", err)
	}
	for _, ok := range []string{"1-2", "c-tom-sky-run"} {
		if err := ValidID(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
}

func TestSaveRejectsInvalidLevels(t *testing.T) {
	_, h, _, local := repoServer(t)
	bad := strings.Replace(goodLevel(t), `"x": 2, "y": 12`, `"x": 999, "y": 12`, 1)
	w := save(h, "local", "c-tom-broken", SaveRequest{Level: bad})
	if w.Code != http.StatusUnprocessableEntity || code(w) != "invalid-level" || !strings.Contains(w.Body.String(), "spawn") {
		t.Errorf("save of a broken level: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(local, "c-tom-broken.json")); err == nil {
		t.Error("a broken level was written")
	}
}

func TestCrossOriginIsRefused(t *testing.T) {
	_, h, builtin, _ := repoServer(t)
	b, _ := json.Marshal(SaveRequest{Level: goodLevel(t)})
	for name, hdr := range map[string]map[string]string{
		"other site":      {"Origin": "https://evil.example"},
		"other port":      {"Origin": "http://127.0.0.1:9999"},
		"no origin":       {},
		"null origin":     {"Origin": "null"},
		"https lookalike": {"Origin": "https://127.0.0.1:41234"},
	} {
		if w := do(h, "PUT", "/api/levels/builtin/c-tom-x", string(b), hdr); w.Code != http.StatusForbidden {
			t.Errorf("%s: got %d", name, w.Code)
		}
	}
	if entries, _ := os.ReadDir(builtin); len(entries) != 1 {
		t.Error("a cross-origin request wrote a file")
	}
}

func TestRebindingHostIsRefused(t *testing.T) {
	_, h, _, _ := repoServer(t)
	r := httptest.NewRequest("GET", "http://evil.example/api/levels", nil)
	r.Host = "evil.example:41234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("a request for another host name got %d", w.Code)
	}
}

func TestOverwriteNeedsConfirming(t *testing.T) {
	_, h, _, local := repoServer(t)
	os.MkdirAll(local, 0o755)
	path := filepath.Join(local, "c-amy-hills.json")
	other := strings.Replace(goodLevel(t), `"timeLimit": 400`, `"timeLimit": 300`, 1)
	os.WriteFile(path, []byte(other), 0o644)

	w := save(h, "local", "c-amy-hills", SaveRequest{Level: goodLevel(t)})
	if w.Code != http.StatusConflict || code(w) != "exists" {
		t.Fatalf("overwriting without confirmation: %d %s", w.Code, w.Body)
	}
	if got, _ := os.ReadFile(path); string(got) != other {
		t.Error("the other level was replaced without confirmation")
	}
	saved := save(h, "local", "c-amy-hills", SaveRequest{Level: goodLevel(t), Overwrite: true, Revision: field(w, "revision")})
	if saved.Code != http.StatusOK {
		t.Fatalf("confirmed overwrite: %d %s", saved.Code, saved.Body)
	}
	// Saving back to the level that was opened never asks.
	if w := save(h, "local", "c-amy-hills", SaveRequest{Level: other, OpenedAs: "local:c-amy-hills", Revision: field(saved, "revision")}); w.Code != http.StatusOK {
		t.Errorf("saving the opened level asked for confirmation: %d", w.Code)
	}
	// The same id in the other folder is a different level: opened as the
	// local one, saving to the built-in folder still needs confirming only
	// if that one exists.
	if w := save(h, "builtin", "c-amy-hills", SaveRequest{Level: other, OpenedAs: "local:c-amy-hills"}); w.Code != http.StatusOK {
		t.Errorf("saving a new built-in copy: %d %s", w.Code, w.Body)
	}
}

func TestSaveRejectsConcurrentEditsFromTheSameRevision(t *testing.T) {
	_, h, _, _ := repoServer(t)
	original := do(h, "GET", "/api/levels/builtin/1-1", "", nil)
	revision := original.Header().Get("ETag")
	if revision == "" {
		t.Fatal("opening the level did not return a revision")
	}
	results := make(chan *httptest.ResponseRecorder, 2)
	data := goodLevel(t)
	for _, timeLimit := range []string{"300", "200"} {
		go func() {
			results <- save(h, "builtin", "1-1", SaveRequest{
				Level:    strings.Replace(data, `"timeLimit": 400`, `"timeLimit": `+timeLimit, 1),
				OpenedAs: "1-1", Revision: revision,
			})
		}()
	}
	success, conflict := 0, 0
	var winningText string
	for range 2 {
		w := <-results
		switch {
		case w.Code == http.StatusOK:
			success++
			winningText = field(w, "text")
		case w.Code == http.StatusConflict && code(w) == "level-changed":
			conflict++
		default:
			t.Errorf("unexpected response: %d %s", w.Code, w.Body)
		}
	}
	current := do(h, "GET", "/api/levels/builtin/1-1", "", nil)
	if success != 1 || conflict != 1 || current.Body.String() != winningText {
		t.Fatalf("successes=%d conflicts=%d saved file matches=%v", success, conflict, current.Body.String() == winningText)
	}
}

func TestListShowsBothFoldersWithSameNames(t *testing.T) {
	_, h, _, local := repoServer(t)
	os.MkdirAll(local, 0o755)
	os.WriteFile(filepath.Join(local, "1-1.json"), []byte(goodLevel(t)), 0o644)
	w := do(h, "GET", "/api/levels", "", nil)
	var list []catalog.Entry
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 2 || list[0].Source != catalog.Builtin || list[1].Source != catalog.Local {
		t.Fatalf("list = %+v", list)
	}
	if list[1].Path == "" {
		t.Error("the list does not say where a level is kept")
	}
	if w := do(h, "GET", "/api/levels/local/1-1", "", nil); w.Code != 200 {
		t.Errorf("read local 1-1: %d", w.Code)
	}
	if w := do(h, "GET", "/api/levels/local/c-nobody-none", "", nil); w.Code != http.StatusNotFound {
		t.Errorf("read missing: %d", w.Code)
	}
}
