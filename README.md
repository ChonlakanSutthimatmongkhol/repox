# 🧠 Repox

**Self-learning scaffold generator that understands your repo's conventions.**

> Generate feature scaffolds that look like your team wrote them — learned from real code, improved by every commit.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)
![Version](https://img.shields.io/badge/Version-v1.0.0-blue?style=flat)

---

## 🎯 What is Repox?

Repox scans your codebase, learns your project's conventions, and generates feature scaffolds that match your team's style. No more copy-pasting from old features. No more writing AI instructions from scratch for every repo.

```bash
# 1. Initialize in your Flutter/Go repo
repox init

# 2. Scan and detect conventions automatically
repox scan

# 3. Generate a feature scaffold matching your conventions
repox generate feature watchlist

# 4. Preview without writing
repox generate feature watchlist --dry-run

# 5. Find similar existing features before generating
repox generate feature watchlist --with-examples

repox generate feature watchlist --ai    # AI-assisted generation (Claude)
repox generate feature watchlist --ai --opus  # Use Claude Opus
repox learn                              # Learn from your edits
repox learn --list                       # Show stored lessons
repox --mcp                              # Start as MCP server (Claude Code / Copilot / Cursor)
```

---

## 📦 Installation

### Prerequisites

- **Go 1.25+** — [install](https://go.dev/dl/)
- **ANTHROPIC_API_KEY** — required for `--ai` and `repox learn` (get one at [console.anthropic.com](https://console.anthropic.com))

### Install via Go

```bash
go install github.com/ChonlakanSutthimatmongkhol/repox/cmd/repox@latest
```

The binary lands at `$(go env GOPATH)/bin/repox`. Make sure that path is in your `$PATH`.

```bash
# Verify installation
repox --version
```

### Build from source

```bash
git clone https://github.com/ChonlakanSutthimatmongkhol/repox.git
cd repox
make install   # builds and copies to $(go env GOPATH)/bin/
```

### Set API key

```bash
# macOS / Linux — add to ~/.zshrc or ~/.bashrc
export ANTHROPIC_API_KEY="sk-ant-..."

# Or set per-project in your IDE launch config
```

---

## 🤖 MCP Setup (Claude Code · Copilot · Cursor)

Repox runs as an MCP stdio server so AI tools can call it directly — no CLI needed.

### Step 1 — Run in your project first

```bash
cd /your/flutter-project
repox init    # creates .repox/
repox scan    # detects conventions → .repox/conventions.json
```

### Step 2 — Add to your AI tool

#### Claude Code (global — all projects)

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "repox": {
      "type": "stdio",
      "command": "/Users/<you>/go/bin/repox",
      "args": ["--mcp"],
      "env": {
        "ANTHROPIC_API_KEY": "sk-ant-..."
      }
    }
  }
}
```

> **Tip:** Claude Code inherits your shell's `$PATH` and env, so if `repox` is already on your PATH and `ANTHROPIC_API_KEY` is exported you can omit the full path and `env` block.

#### Claude Code (project-only)

Create `.claude/mcp.json` at the project root:

```json
{
  "mcpServers": {
    "repox": {
      "type": "stdio",
      "command": "repox",
      "args": ["--mcp"]
    }
  }
}
```

#### GitHub Copilot (VS Code)

Create `.vscode/mcp.json` at the project root:

```json
{
  "servers": {
    "repox": {
      "type": "stdio",
      "command": "repox",
      "args": ["--mcp"]
    }
  }
}
```

Then open **VS Code** → Command Palette → `MCP: List Servers` to confirm it appears.

#### Cursor

Create or edit `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project):

```json
{
  "mcpServers": {
    "repox": {
      "command": "repox",
      "args": ["--mcp"]
    }
  }
}
```

Restart Cursor after saving. The `repox_*` tools should appear in the tool list.

### Step 3 — Ask your AI to use it

Once connected, you can tell your AI assistant:

```
Use repox_scan to detect this project's conventions, then use repox_generate to scaffold a "payments" feature.
```

---

## 🏗️ How It Works

```mermaid
graph LR
    A["👨‍💻 Developer"] -->|"repox generate feature X"| B["🧠 Repox CLI"]
    B --> C["📂 Load Conventions"]
    B --> D["🔍 Find Similar Features"]
    B --> E["🤖 AI Generate"]
    E --> F["📝 Write Files"]
    F --> G["✅ Format & Validate"]
    G --> H["👨‍💻 Developer Reviews"]
    H -->|"commit & repox learn"| I["📚 Lessons Stored"]
    I -->|"next generation"| B

    style A fill:#2d3748,stroke:#4fd1c5,color:#fff
    style B fill:#2d3748,stroke:#f6ad55,color:#fff
    style E fill:#2d3748,stroke:#fc8181,color:#fff
    style I fill:#2d3748,stroke:#68d391,color:#fff
```

---

## 📐 Architecture

```mermaid
graph TB
    subgraph CLI ["⌨️ CLI Layer"]
        CMD_INIT["init"]
        CMD_SCAN["scan"]
        CMD_GEN["generate"]
        CMD_LEARN["learn"]
    end

    subgraph CORE ["⚙️ Core Engine"]
        SCANNER["Scanner"]
        GENERATOR["Generator"]
        RETRIEVER["Example Retriever"]
        LEARNER["Learner"]
    end

    subgraph AI_LAYER ["🤖 AI Layer"]
        PROMPT["Prompt Builder"]
        CLIENT["AI Client"]
        PARSER["Response Parser"]
    end

    subgraph STORAGE ["💾 Storage (.repo-brain/)"]
        CONV["conventions.json"]
        EXAMPLES["examples.json"]
        LESSONS["lessons.json"]
        GENS["generations.json"]
    end

    subgraph EXTERNAL ["🔌 External"]
        CTX["ctx-saver MCP"]
        ANTHROPIC["Anthropic API"]
    end

    CMD_SCAN --> SCANNER
    CMD_GEN --> GENERATOR
    CMD_GEN --> RETRIEVER
    CMD_LEARN --> LEARNER

    SCANNER --> CONV
    RETRIEVER --> EXAMPLES
    LEARNER --> LESSONS
    GENERATOR --> GENS

    GENERATOR --> PROMPT
    PROMPT --> CLIENT
    CLIENT --> ANTHROPIC
    CLIENT --> PARSER

    RETRIEVER --> CTX
    PROMPT --> CTX

    style CLI fill:#1a202c,stroke:#4fd1c5,color:#fff
    style CORE fill:#1a202c,stroke:#f6ad55,color:#fff
    style AI_LAYER fill:#1a202c,stroke:#fc8181,color:#fff
    style STORAGE fill:#1a202c,stroke:#68d391,color:#fff
    style EXTERNAL fill:#1a202c,stroke:#b794f4,color:#fff
```

---

## 🔄 Generation Flow

```mermaid
sequenceDiagram
    actor Dev as Developer
    participant CLI as Repox CLI
    participant Scanner as Scanner
    participant Retriever as Example Retriever
    participant CTX as ctx-saver MCP
    participant AI as AI (Claude)
    participant FS as File System

    Dev->>CLI: repox generate feature watchlist --ai
    CLI->>Scanner: Load conventions.json
    Scanner-->>CLI: Conventions
    CLI->>Retriever: Find similar features
    Retriever->>CTX: Check cached examples
    CTX-->>Retriever: Cached summaries
    Retriever-->>CLI: Top 3 examples
    CLI->>AI: Prompt (conventions + examples + lessons)
    AI-->>CLI: Structured JSON (files + content)
    CLI->>FS: Write generated files
    CLI->>FS: Run formatter
    CLI-->>Dev: ✅ Feature scaffold ready!

    Note over Dev,FS: Developer reviews, edits, commits

    Dev->>CLI: repox learn
    CLI->>FS: Read git diff
    CLI->>AI: Analyze diff → extract lessons
    AI-->>CLI: New lessons
    CLI->>FS: Update lessons.json
    CLI-->>Dev: 📚 Lessons saved for next generation
```

---

## 🚀 Self-Learning Loop

```mermaid
graph TD
    A["🤖 AI Generate Scaffold"] --> B["📝 Developer Edits Code"]
    B --> C["💾 Git Commit"]
    C --> D["🔍 repox learn"]
    D --> E["📊 Diff Analysis"]
    E --> F["📚 Extract Lessons"]
    F --> G{"Dev Approves?"}
    G -->|Yes| H["✅ Save to lessons.json"]
    G -->|No| I["❌ Reject"]
    H --> J["🧠 Better Generation Next Time"]
    J --> A

    style A fill:#4fd1c5,stroke:#2d3748,color:#1a202c
    style D fill:#f6ad55,stroke:#2d3748,color:#1a202c
    style H fill:#68d391,stroke:#2d3748,color:#1a202c
    style J fill:#b794f4,stroke:#2d3748,color:#1a202c
```

---

## 📁 Project Structure

```
repox/
├── cmd/
│   └── repox/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                 # Root command, --version
│   │   ├── init.go                 # repox init
│   │   ├── scan.go                 # repox scan          ✅ v0.2.0
│   │   └── generate.go             # repox generate
│   ├── scanner/                    #                     ✅ v0.2.0
│   │   ├── scanner.go              # Scanner interface
│   │   ├── flutter_scanner.go      # Flutter orchestrator
│   │   ├── go_scanner.go           # Go scanner (stub, v0.3.0)
│   │   ├── project_detector.go     # flutter/go/node detection
│   │   ├── folder_scanner.go       # Feature root & structure
│   │   ├── naming_scanner.go       # Suffix detection (majority vote)
│   │   ├── import_scanner.go       # Common import ranking
│   │   └── routing_scanner.go      # State mgmt & routing
│   ├── generator/
│   │   ├── naming.go               # ToSnakeCase, ToPascalCase, ToCamelCase
│   │   ├── template_generator.go   # embed.FS template rendering
│   │   └── file_writer.go          # Safe file writer
│   ├── retriever/                  #                     ✅ v0.3.0
│   │   ├── retriever.go            # Retriever interface
│   │   ├── feature_indexer.go      # Walk repo and index features
│   │   ├── similarity.go           # Score similarity between features
│   │   └── retriever_test.go       # Tests
│   ├── learner/                    #                     ✅ v0.5.0
│   │   ├── learner.go              # Learner interface
│   │   ├── diff_reader.go          # Compare snapshots vs current files
│   │   ├── lesson_extractor.go     # AI-powered lesson extraction
│   │   └── learner_test.go         # Tests
│   ├── ai/                         #                     ✅ v0.4.0
│   │   ├── client.go               # Client + Caller interfaces
│   │   ├── anthropic.go            # Anthropic API (raw HTTP)
│   │   ├── prompt_builder.go       # Prompt assembly with token budget
│   │   ├── response_parser.go      # JSON response parser
│   │   └── ai_test.go              # Tests
│   ├── mcp/                        #                     ✅ v1.0.0
│   │   ├── server.go               # MCP server setup (mark3labs/mcp-go)
│   │   ├── tools.go                # Tool schemas
│   │   ├── handlers.go             # Tool handlers
│   │   └── mcp_test.go             # Tests
│   ├── config/
│   │   └── loader.go               # Generic Load[T]/Save[T], defaults
│   └── models/
│       ├── convention.go           # Convention, NamingConvention, RoutingConfig
│       ├── config.go               # Config, AIConfig
│       ├── example.go              # Example
│       ├── lesson.go               # Lesson
│       └── generation.go           # Generation log
├── templates/
│   └── flutter_bloc_feature/
│       ├── screen.dart.tmpl
│       ├── bloc.dart.tmpl
│       ├── event.dart.tmpl
│       ├── state.dart.tmpl
│       ├── repository.dart.tmpl
│       ├── repository_impl.dart.tmpl
│       ├── usecase.dart.tmpl
│       ├── request.dart.tmpl
│       ├── response.dart.tmpl
│       └── bloc_test.dart.tmpl
├── .repox/                         # Created by `repox init`
│   ├── config.json
│   ├── conventions.json
│   ├── examples.json
│   ├── lessons.json
│   └── generations.json
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## ⚙️ Configuration

### `.repox/config.json`

```json
{
  "version": "0.1.0",
  "project_type": "flutter",
  "feature_root": "lib/features",
  "test_root": "test/features",
  "default_template": "flutter_bloc_feature",
  "ai": {
    "provider": "anthropic",
    "generation_model": "claude-sonnet-4-20250514",
    "learning_model": "claude-haiku-4-5-20251001"
  }
}
```

### `.repox/conventions.json`

```json
{
  "project_type": "flutter",
  "state_management": "flutter_bloc",
  "feature_structure": "clean_architecture",
  "feature_root": "lib/features",
  "test_root": "test/features",
  "naming": {
    "class_case": "PascalCase",
    "file_case": "snake_case",
    "screen_suffix": "Screen",
    "bloc_suffix": "Bloc",
    "event_suffix": "Event",
    "state_suffix": "State",
    "repository_suffix": "Repository",
    "usecase_suffix": "UseCase"
  },
  "routing": {
    "type": "go_router",
    "route_file": "lib/router/app_router.dart"
  },
  "common_imports": [
    "package:flutter/material.dart",
    "package:flutter_bloc/flutter_bloc.dart"
  ]
}
```

---

## 📦 Usage Examples

### `repox scan`

```bash
$ repox scan
Scanned repository:
  Project type:      flutter
  State management:  flutter_bloc
  Feature root:      lib/features
  Feature structure: clean_architecture
  Test root:         test/features
  Naming:
    Screen suffix:   Screen
    Bloc suffix:     Bloc
    Event suffix:    Event
    State suffix:    State
  Routing:           go_router (lib/router/app_router.dart)
  Common imports:    5 detected

Conventions saved to .repox/conventions.json
```

### `repox generate feature payments --with-examples`

```
Similar features found:
  - home (lib/features/home)
  - profile (lib/features/profile)

  created lib/features/payments/payments_bloc.dart
  ...
```

### `repox generate feature watchlist`

```
  created lib/features/watchlist/watchlist_bloc.dart
  created lib/features/watchlist/watchlist_event.dart
  created lib/features/watchlist/watchlist_state.dart
  created lib/features/watchlist/watchlist_screen.dart
  created lib/features/watchlist/watchlist_repository.dart
  created lib/features/watchlist/watchlist_repository_impl.dart
  created lib/features/watchlist/watchlist_usecase.dart
  created lib/features/watchlist/watchlist_request.dart
  created lib/features/watchlist/watchlist_response.dart
  created test/features/watchlist/watchlist_bloc_test.dart

10 created, 0 skipped
```

---

## 🛡️ Safety

- Never sends secrets (`.env`, keys, certificates) to AI
- `--dry-run` mode to preview before writing
- No overwrite without `--force`
- Path allowlist/denylist support
- Local-only mode available
- Audit log of all generations

---

## 🗺️ Roadmap

### ✅ v0.1.0 — Template Generator MVP _(released 2026-05-09)_

> Goal: `repox generate feature watchlist` ทำงานได้โดยไม่ต้องใช้ AI

- ✅ CLI skeleton (init, generate)
- ✅ Go `text/template` engine with `embed.FS`
- ✅ Flutter BLoC feature template (10 files)
- ✅ Naming conversion (snake_case, PascalCase, camelCase)
- ✅ File writer with `--force` / `--dry-run` protection
- ✅ Config: `.repox/config.json`

### ✅ v0.2.0 — Repo Scanner _(released 2026-05-09)_

> Goal: `repox scan` อ่าน repo แล้วสร้าง conventions.json อัตโนมัติ

- ✅ Detect project type (Flutter, Go, Node)
- ✅ Detect feature root & folder structure (clean_architecture / grouped / flat)
- ✅ Detect state management (BLoC, Cubit, Riverpod, Provider)
- ✅ Detect naming conventions from existing files (majority vote)
- ✅ Detect routing (go_router, auto_route) + route file path
- ✅ Detect common imports (top 10 by frequency)
- ✅ Write `.repox/conventions.json`
- ✅ `repox generate` uses scanned conventions automatically

### ✅ v0.3.0 — Example Retrieval _(released 2026-05-09)_

> Goal: `repox generate feature X --with-examples` ใช้ feature เก่าเป็นต้นแบบ

- ✅ Index existing features in repo (`internal/retriever/`)
- ✅ Feature metadata extraction (components, imports, patterns)
- ✅ Similarity scoring (name 0.2 + structure 0.3 + imports 0.2 + patterns 0.3)
- ✅ Top-N example selection (`FindSimilar`)
- ✅ `repox scan` indexes features → `.repox/examples.json`
- ✅ `--with-examples` flag prints similar features before generating

### v0.4.0 — AI-Assisted Generation

> Goal: `repox generate feature X --ai` ให้ AI generate code ตาม convention จริง

- AI provider abstraction (Anthropic as default)
- Prompt builder (conventions + examples + lessons)
- Structured JSON response parsing
- Token budget control via ctx-saver
- Model selection: Sonnet (default) / Opus (`--opus`)
- Formatter & analyzer integration
- `--diff` mode to preview changes

### v0.5.0 — Self-Learning Loop

> Goal: `repox learn` เรียนรู้จาก diff หลัง dev แก้ code

- Store generation snapshots
- Git diff reader (generated vs committed)
- Lesson extraction via AI (Haiku)
- Lesson approve / reject workflow
- Confidence scoring & dedup
- Inject lessons into next generation prompt

### ✅ v1.0.0 — MCP Server Mode _(released 2026-05-09)_

> Goal: AI tools เรียก Repox ผ่าน MCP ได้ตรง ไม่ต้องผ่าน CLI

- ✅ MCP server via `repox --mcp` (stdio JSON-RPC)
- ✅ 5 tools: `repox_scan`, `repox_generate`, `repox_find_similar`, `repox_learn`, `repox_explain_convention`
- ✅ Works with: Claude Code, GitHub Copilot, Cursor

### v1.x — Future

- Go backend template (handler / service / repository)
- Jira ticket → scaffold
- API schema → DTO / service / repository
- VS Code extension
- Team convention sharing
- Architecture drift detector

---

## 🔌 MCP Tool Reference

> Full setup instructions are in the [MCP Setup](#-mcp-setup-claude-code--copilot--cursor) section above.

| Tool | Parameters | Description |
|---|---|---|
| `repox_scan` | `project_override?` | Scan repo, detect conventions, index features → saves `.repox/conventions.json` |
| `repox_generate` | `feature_name`, `use_ai?`, `use_examples?`, `force?`, `dry_run?` | Generate feature scaffold (template or AI mode) |
| `repox_find_similar` | `feature_name`, `top_n?` | Find structurally similar existing features (default top 3) |
| `repox_learn` | `generation_id?`, `auto_approve?` | Returns CLI usage hint — run `repox learn` in terminal for full flow |
| `repox_explain_convention` | — | Return repo conventions in human-readable format |

Works with: **Claude Code** · **GitHub Copilot (VS Code)** · **Cursor**

---

## 📄 License

MIT

---

<p align="center">
  <b>Repox</b> — Your repo already knows its conventions. Let it teach the AI.
</p>
