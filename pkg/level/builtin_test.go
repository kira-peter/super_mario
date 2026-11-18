package level

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// checkBuiltin is what the repository test runs on one level file: it must
// load, and it must already be in the canonical layout. The error names the
// level and, for a load failure, the loader's own row and column.
func checkBuiltin(name string, data []byte) error {
	if _, err := Parse(data); err != nil {
		return fmt.Errorf("关卡 %s 无法加载：%w", name, err)
	}
	formatted, err := Format(data)
	if err != nil {
		return fmt.Errorf("关卡 %s 无法规整：%w", name, err)
	}
	if !bytes.Equal(data, formatted) {
		if bytes.Contains(data, []byte("\r\n")) {
			return fmt.Errorf("关卡 %s 用的是 CRLF 换行，关卡文件必须是 LF；仓库的 .gitattributes 会处理，重新检出文件（git checkout -- pkg/level/data）即可", name)
		}
		line := firstDiffLine(data, formatted)
		return fmt.Errorf("关卡 %s 的排版与编辑器保存的不一致（第 %d 行起）；用编辑器重新保存，或运行 `go run ./cmd/super-mario fmt-levels`", name, line)
	}
	return nil
}

func firstDiffLine(a, b []byte) int {
	la := strings.Split(string(a), "\n")
	lb := strings.Split(string(b), "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return i + 1
		}
	}
	if len(la) < len(lb) {
		return len(la) + 1
	}
	return len(lb) + 1
}

// TestEveryBuiltinLevelIsValid is the gate a contributed level has to pass to
// get into the game (spec level-contribution / 仓库关卡自动校验).
func TestEveryBuiltinLevelIsValid(t *testing.T) {
	names, err := EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no built-in levels found")
	}
	for _, name := range names {
		data, err := assets.ReadFile("data/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := checkBuiltin(name, data); err != nil {
			t.Error(err)
		}
	}
}

func TestBuiltinCheckNamesTheBadRow(t *testing.T) {
	bad := []byte(`{"name":"t","world":"1-1","width":4,"height":3,"timeLimit":400,
	"spawn":{"x":0,"y":1},"flag":{"x":3,"topY":0,"bottomY":1},
	"rows":["    ","   ","XXXX"],"contents":[],"entities":[]}`)
	err := checkBuiltin("c-tom-bad", bad)
	if err == nil {
		t.Fatal("a level with a short row passed")
	}
	for _, want := range []string{"c-tom-bad", "row 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestBuiltinCheckCatchesLayoutDrift(t *testing.T) {
	good, err := assets.ReadFile("data/1-1.json")
	if err != nil {
		t.Fatal(err)
	}
	// Re-indent one line the way a hand edit might.
	drifted := bytes.Replace(good, []byte("\n  \"width\""), []byte("\n    \"width\""), 1)
	err = checkBuiltin("1-1", drifted)
	if err == nil {
		t.Fatal("a re-indented level passed")
	}
	if !strings.Contains(err.Error(), "fmt-levels") || !strings.Contains(err.Error(), "第 4 行") {
		t.Errorf("error %q should say where and how to fix it", err)
	}
}
