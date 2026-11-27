package editor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"super-mario/pkg/catalog"
)

func TestDraftSurvivesTheEditorChangingPorts(t *testing.T) {
	config := Config{Catalog: &catalog.Catalog{LocalDir: t.TempDir()}, DraftDir: t.TempDir()}
	first, err := Start(config)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Stop()
	const token = "3ce559c2-c4f7-4296-ada5-76787e85dbaf"
	const key = "local:c-amy-hills"
	const revision = "loaded-version"
	const text = "an incomplete level can still be recovered"
	raw, err := json.Marshal(draft{OpenedAs: key, Revision: revision, Level: text})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, first.URL+"api/drafts/"+token, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", first.URL[:len(first.URL)-1])
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("write draft: %d", resp.StatusCode)
	}

	second, err := Start(config)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Stop()
	if err := first.Stop(); err != nil {
		t.Fatal(err)
	}
	if first.URL == second.URL {
		t.Fatal("expected two different loopback origins")
	}
	resp, err = http.Get(second.URL + "api/drafts?key=" + key)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var saved draft
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Token != token || saved.OpenedAs != key || saved.Revision != revision || saved.Level != text {
		t.Fatalf("wrong recovered draft: %+v", saved)
	}
}
