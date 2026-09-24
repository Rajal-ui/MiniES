# OpenCode — Implementation Plan
> **D:/ Drive Only · Python venv Enabled**
> All tooling installed on `D:\` to keep `C:\` clean.

---

## Pre-Setup — Redirect Everything to D:/

### Step 1 — Create D:/ Tool Directories

```powershell
New-Item -ItemType Directory -Force -Path "D:\Tools\npm-global"
New-Item -ItemType Directory -Force -Path "D:\Tools\npm-cache"
New-Item -ItemType Directory -Force -Path "D:\Tools\opencode"
```

### Step 2 — Redirect npm Prefix & Cache

```powershell
npm config set prefix "D:\Tools\npm-global"
npm config set cache  "D:\Tools\npm-cache"
```

### Step 3 — Add to PowerShell Profile (persists across sessions)

```powershell
$p = $PROFILE
if (-not (Test-Path $p)) { New-Item -Force $p | Out-Null }
Add-Content $p "`n`$env:PATH = 'D:\Tools\npm-global;' + `$env:PATH"
Add-Content $p "`$env:NPM_CONFIG_PREFIX = 'D:\Tools\npm-global'"
Add-Content $p "`$env:NPM_CONFIG_CACHE  = 'D:\Tools\npm-cache'"
Add-Content $p "`$env:OPENCODE_CONFIG_HOME = 'D:\Tools\opencode'"

# Apply immediately
$env:PATH = "D:\Tools\npm-global;" + $env:PATH
$env:OPENCODE_CONFIG_HOME = "D:\Tools\opencode"
```

---

## Phase 1 — Install OpenCode on D:/

With npm prefix pointing to `D:\Tools\npm-global`:

```powershell
npm install -g opencode-ai
```

Binary lands at: `D:\Tools\npm-global\opencode.cmd`

```powershell
# Verify
opencode --version       # 1.18.32
where.exe opencode       # D:\Tools\npm-global\opencode.cmd
```

> NOTE: opencode v1.18.32 is already on your system (C:/). Re-install after
> redirecting prefix to move the binary to D:/.

---

## Phase 2 — Python venv on D:/

### 2.1 Create & Activate

```powershell
cd D:\Projects\MiniES

python -m venv .venv              # creates D:\Projects\MiniES\.venv
.\.venv\Scripts\Activate.ps1     # activate (prompt shows (.venv))
```

### 2.2 Upgrade pip

```powershell
python -m pip install --upgrade pip
```

### 2.3 Install Python Language Server (pylsp) inside venv

```powershell
pip install "python-lsp-server[all]"
pip install pylsp-mypy           # type checking
pip install python-lsp-ruff      # fast linting
pip install ruff mypy pytest     # dev tools
```

```powershell
# Verify
pylsp --version
where.exe pylsp   # D:\Projects\MiniES\.venv\Scripts\pylsp.exe
```

### 2.4 Save requirements

```powershell
pip freeze > requirements.txt
```

### 2.5 Auto-activate venv (optional — add to $PROFILE)

```powershell
function Set-AutoVenv {
    $v = Join-Path (Get-Location) ".venv\Scripts\Activate.ps1"
    if (Test-Path $v) { & $v }
}
```

---

## Phase 3 — Configure opencode (D:\Tools\opencode\config.json)

Create `D:\Tools\opencode\config.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-sonnet-4-5",
  "providers": {
    "anthropic": { "apiKey": "${ANTHROPIC_API_KEY}" },
    "openai":    { "apiKey": "${OPENAI_API_KEY}" },
    "google":    { "apiKey": "${GOOGLE_API_KEY}" }
  },
  "permissions": {
    "bash":  "ask",
    "write": "always",
    "fetch": "ask"
  },
  "lsp": {
    "python": {
      "command": "D:\\Projects\\MiniES\\.venv\\Scripts\\pylsp.exe"
    }
  },
  "theme": "opencode"
}
```

Store API keys as User env vars (never hardcode):

```powershell
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_API_KEY", "sk-ant-...", "User")
[System.Environment]::SetEnvironmentVariable("OPENAI_API_KEY", "sk-...", "User")
```

### Provider Recommendations

| Use Case         | Provider  | Model                 |
|------------------|-----------|-----------------------|
| Best quality     | Anthropic | `claude-sonnet-4-5`   |
| Cost-effective   | Google    | `gemini-1.5-pro`      |
| Local / private  | Ollama    | `llama3.2`            |
| Fast iteration   | OpenAI    | `gpt-4o-mini`         |

---

## Phase 4 — Project Setup

### Launch opencode in MiniES

```powershell
cd D:\Projects\MiniES
.\.venv\Scripts\Activate.ps1
opencode
```

### Create AGENTS.md (project root)

```markdown
# Project: MiniES

## Tech Stack
- Python 3.13
- venv at `.venv/`

## Conventions
- Type hints on all functions
- PEP 8 style
- Docstrings on all public functions
- Tests in /tests/

## Commands
- Run tests:  pytest tests/ -v
- Lint:       ruff check .
- Type check: mypy .
```

> Place at `D:\Projects\MiniES\AGENTS.md` — opencode reads it every session.

---

## Phase 5 — Core Usage

### Launch Modes

| Mode        | Command                              |
|-------------|--------------------------------------|
| TUI         | `opencode`                           |
| Web UI      | `opencode --web`                     |
| CLI one-shot| `opencode run "fix bug in main.py"`  |
| Resume      | `opencode -s ses_f2da7c57...`        |

### Key TUI Keybinds

| Shortcut  | Action                    |
|-----------|---------------------------|
| `Ctrl+N`  | New session               |
| `Ctrl+S`  | Share session             |
| `Ctrl+T`  | Toggle theme              |
| `Ctrl+/`  | Show keybind help         |
| `Esc`     | Cancel current operation  |

---

## Phase 6 — Custom Commands (config.json)

```json
{
  "commands": {
    "test": {
      "description": "Run test suite",
      "command": "D:\\Projects\\MiniES\\.venv\\Scripts\\pytest.exe tests/ -v"
    },
    "lint": {
      "description": "Lint with ruff",
      "command": "D:\\Projects\\MiniES\\.venv\\Scripts\\ruff.exe check ."
    },
    "typecheck": {
      "description": "Type check with mypy",
      "command": "D:\\Projects\\MiniES\\.venv\\Scripts\\mypy.exe ."
    }
  }
}
```

Use in TUI: `/test`, `/lint`, `/typecheck`

---

## Phase 7 — MCP Server Integration

```json
{
  "mcp": {
    "servers": {
      "github": {
        "type": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-github"],
        "env": { "GITHUB_TOKEN": "${GITHUB_TOKEN}" }
      }
    }
  }
}
```

---

## Phase 8 — Session Management

```powershell
opencode -s minies-dev              # start named session
opencode sessions list              # list all sessions
opencode -s ses_f2da7c57...         # resume existing session
```

---

## Phase 9 — Advanced Agents

```json
{
  "agents": {
    "reviewer": {
      "model": "anthropic/claude-opus-4",
      "system": "Strict code reviewer. Focus on security and correctness."
    }
  }
}
```

---

## Implementation Checklist

### D:/ Environment
- [ ] `D:\Tools\npm-global` created + `npm config set prefix`
- [ ] `D:\Tools\npm-cache` created  + `npm config set cache`
- [ ] `D:\Tools\opencode` created   + `OPENCODE_CONFIG_HOME` set
- [ ] All env vars added to `$PROFILE`
- [ ] `D:\Tools\npm-global` in user PATH

### OpenCode
- [ ] `npm install -g opencode-ai` (lands on D:/)
- [ ] `where.exe opencode` → `D:\Tools\npm-global\opencode.cmd`
- [ ] `D:\Tools\opencode\config.json` created

### Python venv
- [ ] `python -m venv D:\Projects\MiniES\.venv`
- [ ] Activated: `.\.venv\Scripts\Activate.ps1`
- [ ] Installed: `python-lsp-server[all]` + `pylsp-mypy` + `python-lsp-ruff`
- [ ] `requirements.txt` saved
- [ ] pylsp path in opencode config.json

### Project
- [ ] `AGENTS.md` at `D:\Projects\MiniES\AGENTS.md`
- [ ] Custom commands: `/test` `/lint` `/typecheck`
- [ ] Session naming convention adopted
