# 提交版本

每次提交都要根据该提交的实际改动递增 `VERSION`，并在同一个提交中同步 `pkg/version/version.go` 的 `Default`。使用 `go run ./tools/bumpversion <patch|minor|major> -w` 更新这两个文件。

- `major` 用于不兼容的改动。
- `minor` 用于兼容的新功能。
- `patch` 用于修复、重构、测试、文档和构建维护。

同一提交包含多类改动时，按影响最大的一类升版。重新构建 exe 后，运行 `./build/super-mario.exe version` 核对版本。详细规则见 [VERSIONING.md](VERSIONING.md)。
