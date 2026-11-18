package level

import (
	"strings"
	"testing"
)

// TestCheckReportsEveryProblem: one level with several mistakes gets all of
// them back at once, each with the cell it is about.
func TestCheckReportsEveryProblem(t *testing.T) {
	data := []byte(`{"name":"t","world":"1-1","width":5,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":4,"topY":0,"bottomY":1},
	"rows":["?  Z?","     ","XXXXX"],
	"contents":[{"x":4,"y":0,"kind":"fireflower"}],
	"entities":[{"kind":"bowser","x":2,"y":1}]}`)
	probs := Check(data)

	want := map[string][2]int{
		"symbol":      {3, 0},
		"block-kind":  {4, 0},
		"entity-kind": {2, 1},
		"block-empty": {0, 0},
	}
	got := map[string][2]int{}
	for _, p := range probs {
		got[p.Code] = [2]int{p.X, p.Y}
	}
	for code, at := range want {
		g, ok := got[code]
		if !ok {
			t.Errorf("no %q problem reported; got %+v", code, probs)
			continue
		}
		if g != at {
			t.Errorf("%q reported at %v, want %v", code, g, at)
		}
	}

	// Parse agrees, and reports the first of them.
	_, err := Parse(data)
	if err == nil || err.Error() != probs[0].Message {
		t.Errorf("Parse error %v, want the first problem %q", err, probs[0].Message)
	}
}

func TestCheckIsEmptyForAGoodLevel(t *testing.T) {
	data, _ := assets.ReadFile("data/1-1.json")
	if probs := Check(data); len(probs) != 0 {
		t.Errorf("1-1 has problems: %+v", probs)
	}
}

func TestCheckReportsBadJSON(t *testing.T) {
	probs := Check([]byte(`{"name":`))
	if len(probs) != 1 || probs[0].Code != "json" {
		t.Errorf("bad json: %+v", probs)
	}
	probs = Check([]byte(`{"name":"t","contnets":[]}`))
	if len(probs) != 1 || !strings.Contains(probs[0].Message, "contnets") {
		t.Errorf("a misspelt field was not named: %+v", probs)
	}
}

// TestBrokenGridStopsCellChecks: when a row is the wrong width, cell positions
// are meaningless, so block checks do not pile on misleading errors.
func TestBrokenGridStopsCellChecks(t *testing.T) {
	probs := Check([]byte(`{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["?   ","   ","XXXX"],"contents":[],"entities":[]}`))
	for _, p := range probs {
		if p.Code == "block-empty" {
			t.Errorf("reported %q although the grid is broken", p.Message)
		}
	}
	if len(probs) != 1 || probs[0].Code != "row-width" || probs[0].Y != 1 {
		t.Errorf("want just the short row: %+v", probs)
	}
}
