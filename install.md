# install.md

本文件是给 Agent 用的安装与环境检查规范。
目标：先检查，再征求同意，最后自动安装（重点是 Go 环境）。

## 1) 环境要求

最低要求：

- OS：Linux / macOS / WSL2
- Shell：`bash`
- 基础工具：`curl`、`tar`、`git`
- Node.js：`>= 20`（用于浏览器 worker / 脚本）
- Go：`>= 1.22`（项目 `go.mod` 要求）

## 2) Agent 执行顺序（强制）

1. 先检测环境，不直接安装。
2. 若环境不符合，明确列出不符合项。
3. 向用户征求意见（必须得到“同意安装”）。
4. 用户同意后，自动安装缺失环境。
5. 安装完成后再次检测并回报结果。

## 3) 检测命令（建议）

```bash
set -e

echo "[Check] OS: $(uname -s)"
command -v bash >/dev/null && echo "[OK] bash" || echo "[MISS] bash"
command -v curl >/dev/null && echo "[OK] curl" || echo "[MISS] curl"
command -v tar  >/dev/null && echo "[OK] tar"  || echo "[MISS] tar"
command -v git  >/dev/null && echo "[OK] git"  || echo "[MISS] git"

if command -v node >/dev/null; then
  echo "[OK] node $(node -v)"
else
  echo "[MISS] node"
fi

if command -v go >/dev/null; then
  echo "[OK] go $(go version)"
else
  echo "[MISS] go"
fi
```

## 4) 询问模板（环境不符合时）

Agent 必须先发确认：

```text
检测到环境不符合：
1) Go 未安装 / 版本低于 1.22
2) （如有）Node 未安装或版本过低

是否同意我自动安装缺失环境？
回复“同意”后我将开始安装，并在完成后回报版本结果。
```

## 5) 自动安装（用户同意后）

### 5.1 Go 自动安装（主路径）

优先安装 Go 1.22.x（或更新稳定版），并写入 PATH。

Linux（x86_64）示例：

```bash
set -e
GO_VERSION="1.22.12"
ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
URL="https://go.dev/dl/${ARCHIVE}"

curl -fL "$URL" -o "/tmp/${ARCHIVE}"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "/tmp/${ARCHIVE}"

if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
  echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
fi

export PATH=/usr/local/go/bin:$PATH
go version
```

macOS（Apple Silicon 可改 arm64 包）示例：

```bash
set -e
GO_VERSION="1.22.12"
ARCHIVE="go${GO_VERSION}.darwin-amd64.tar.gz" # Apple Silicon 改为 darwin-arm64
URL="https://go.dev/dl/${ARCHIVE}"

curl -fL "$URL" -o "/tmp/${ARCHIVE}"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "/tmp/${ARCHIVE}"

if ! grep -q '/usr/local/go/bin' ~/.zshrc; then
  echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.zshrc
fi

export PATH=/usr/local/go/bin:$PATH
go version
```

## 6) 安装后验证

```bash
go version
node -v
cd dy-ks-mcp
go mod tidy
go test ./...
go build ./...
```

## 7) 失败处理

- 下载失败：检查网络或更换镜像后重试。
- 权限不足：提示用户授权 `sudo` 再继续。
- 版本仍不满足：回报实际版本，并再次征求是否继续处理。

---

执行原则：
- 不经同意不安装。
- 同意后自动安装。
- 全程可追踪、可复现、可回滚。
