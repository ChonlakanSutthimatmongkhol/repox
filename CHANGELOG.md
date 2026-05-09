# Changelog

All notable changes to Repox are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [1.0.3] — 2026-05-10

### Added

- Scan feature file anatomy per role, including class names, base classes, methods, constructor dependencies, imports, and capabilities such as `firebase_tracking`, `analytics`, `base_bloc`, and `route_model`.
- Summarize role anatomy in `.repox/conventions.json`, `repox scan`, and generated project skills.
- Add `repox plan feature <name-or-path>` to preview files, selected roles, and anatomy hints before generation.
- Add `repox generate feature --roles <roles>` for role-filtered generation.
- Add `repox generate feature --like <existing-feature>` to reuse an existing feature's shape, roles, routes, and structure.

### Changed

- Version bumped to **1.0.3**.

---

## [1.0.2] — 2026-05-09

### Added

- Detect nested feature flow folders such as `lib/features/investment/fund_list`.
- Store per-feature role files and file routes in `.repox/conventions.json`.
- Generate slash-separated feature paths using the leaf feature name for class and file names.

### Changed

- `repox scan` now skips implementation-only folders such as `firebase`, `analytics`, and empty container folders.
- `repox generate feature <name-or-path>` uses exact scanned feature routes before falling back to global project pattern mappings.
- Version bumped to **1.0.2**.

---

## [1.0.0] — 2026-05-09

### Added

- **`repox --mcp`** — starts Repox as an MCP (Model Context Protocol) stdio server; works with Claude Code, GitHub Copilot, and Cursor
- **`internal/mcp` package** — server setup, tool definitions, and handlers using `github.com/mark3labs/mcp-go`
- **5 MCP tools exposed:**
  - `repox_scan` — scan repo, detect conventions, index features
  - `repox_generate` — generate feature scaffold (supports `use_ai`, `use_examples`, `force`, `dry_run`)
  - `repox_find_similar` — find structurally similar existing features
  - `repox_learn` — delegates to CLI with usage hint
  - `repox_explain_convention` — returns full convention summary in natural language
- All existing CLI commands (`init`, `scan`, `generate`, `learn`) remain unchanged

### Changed

- Go module version bumped to 1.25.5 (required by mcp-go v0.52.0)
- Version bumped to **1.0.0**

---

## [0.5.0] — 2026-05-09

### Added

- **`repox learn` command** — extracts reusable lessons from AI-generated vs developer-edited diffs
  - Flags: `--from <gen_id>`, `--approve`, `--reject <id>`, `--list`, `--prune`, `--reset`
- **`internal/learner` package** — `ReadDiffs`, `ExtractLessons`, `ParseExtractionJSON`
- **Snapshot storage** — AI-mode generations copy file contents to `.repox/snapshots/<gen_id>/`
- **Lesson deduplication** — word-overlap similarity prevents near-duplicate lessons
- **Lesson injection** — approved lessons filtered by scope and confidence injected into next AI prompt
- **`ai.Caller` interface** — lower-level `Call(system, user string) (string, error)` for custom prompts
- **Updated `models.Lesson`** — full struct: `ID`, `Scope`, `Lesson`, `Confidence`, `Approved`, `Source`
- **Updated `models.Generation`** — added `Mode` ("template"/"ai") and `SnapshotDir` fields

---

## [0.4.0] — 2026-05-09

### Added

- **`repox generate --ai`** — sends conventions + examples + lessons to Claude API and writes AI-generated files
- **`repox generate --opus`** — uses `claude-opus-4-7` instead of default Sonnet
- **`internal/ai` package** — `Client`/`Caller` interfaces, `AnthropicClient` (raw HTTP, no SDK), `BuildSystemPrompt`, `BuildUserPrompt`, `ParseResponse`
- **Token budget control** — examples capped at 3 features × 2 files × 100 lines; total budget 30K tokens
- **Generation logging** — appended to `.repox/generations.json` after every generate (template or AI)
- **`dart format` integration** — runs automatically after writing `.dart` files; skipped if dart not installed
- **Error messages** — clear guidance for missing API key, API errors, invalid JSON response

---

## [0.3.0] — 2026-05-09

### Added

- **`internal/retriever` package** — indexes existing features and ranks them by similarity
- **`IndexFeatures(rootDir, conv)`** — walks `featureRoot` first-level subdirectories and builds `Example` metadata for each feature (components, imports, patterns)
- **`ScoreSimilarity(target, example)`** — weighted similarity score (name overlap 0.2 + component structure 0.3 + imports 0.2 + patterns 0.3)
- **`FindSimilar(target, examples, topN)`** — returns top-N most similar features sorted by score descending
- **`FeatureIndexer` struct** implements `Retriever` interface (`Index` + `FindSimilar`)
- **`--with-examples` flag** on `repox generate feature` — loads (or re-indexes) examples, prints top 3 similar features before generating
- **`repox scan` now indexes features** — runs `IndexFeatures` after convention detection and saves `.repox/examples.json`; prints `"Indexed N features"`
- **Updated `models.Example`** — replaced stub with full struct: `Name`, `Path`, `Files` (role map), `Patterns`, `Metadata`
- **`models.FeatureMetadata`** — `HasBloc`, `HasScreen`, `HasRepository`, `HasUseCase`, `HasTest`, `Imports`, `Structure`

---

## [0.2.0] — 2026-05-09

### Added

- **`repox scan` command** — analyzes the current repository and writes detected conventions to `.repox/conventions.json`
- **Project type detection** — detects `flutter`, `dart`, `go`, or `node` from marker files (`pubspec.yaml`, `go.mod`, `package.json`)
- **Feature root detection** — locates `lib/features`, `lib/modules`, `lib/pages`, `lib/screens`, or `internal/` automatically
- **Folder structure detection** — classifies projects as `clean_architecture` (presentation+domain layers), `grouped` (bloc/screen/repository dirs), or `flat`
- **Naming convention detection** — majority-vote analysis of existing `.dart` files to infer screen, bloc, event, state, repository, and usecase suffixes
- **Common import detection** — ranks the top 10 most-used package imports across the feature tree
- **State management detection** — reads `pubspec.yaml` to detect `flutter_bloc`, `riverpod`, or `provider`
- **Routing detection** — detects `go_router` or `auto_route` from dependencies; finds the route file path automatically
- **`FlutterScanner`** orchestrator in `internal/scanner/flutter_scanner.go`
- **`GoScanner`** stub in `internal/scanner/go_scanner.go` (full implementation coming in v0.3.0)
- **`--project` flag** on `repox scan` to override project type detection
- **`--deep` flag** on `repox scan` (default: `true`) to include file-content scanning for imports
- `gopkg.in/yaml.v3` dependency for robust `pubspec.yaml` parsing
- `repox generate` now reads the scanned `conventions.json` — generated files follow real repo conventions instead of defaults

### Changed

- `repox generate feature <name>` uses scanned conventions from `.repox/conventions.json` (feature root, naming suffixes) instead of hardcoded defaults
- `repox scan` also updates `project_type` and `feature_root` in `.repox/config.json`

---

## [0.1.0] — 2026-05-09

### Added

- **`repox init` command** — creates `.repox/` directory with `config.json`, `conventions.json`, `examples.json`, `lessons.json`, `generations.json`
- **`repox generate feature <name>` command** — generates a full Flutter BLoC feature scaffold from embedded templates
  - Flags: `--force` (overwrite), `--dry-run` (preview), `--template` (override template name)
  - Colorized output: green for created files, yellow for skipped
- **Flutter BLoC templates** (`templates/flutter_bloc_feature/`):
  - `screen.dart.tmpl`, `bloc.dart.tmpl`, `event.dart.tmpl`, `state.dart.tmpl`
  - `repository.dart.tmpl`, `repository_impl.dart.tmpl`, `usecase.dart.tmpl`
  - `request.dart.tmpl`, `response.dart.tmpl`, `bloc_test.dart.tmpl`
- **Naming utilities** (`internal/generator/naming.go`):
  - `ToSnakeCase` — `WatchList` → `watch_list`
  - `ToPascalCase` — `watch-list` → `WatchList`
  - `ToCamelCase` — `watch_list` → `watchList`
- **Generic JSON config loader** (`internal/config/loader.go`) — `Load[T]`, `Save[T]`, `RepoxPath`, `RepoxDirExists`, `DefaultConfig`, `DefaultConventions`
- **File writer** with overwrite protection — refuses to overwrite without `--force`; prints `"File exists: <path>. Use --force to overwrite"`
- **Template engine** using `embed.FS` — templates compiled into the binary at build time
- **Data models** in `internal/models/`: `Config`, `Convention`, `NamingConvention`, `RoutingConfig`, `Generation`, `Example`, `Lesson`
- **`--version` flag** — prints `repox v0.1.0`
- **Makefile** with `build`, `test`, `install`, `lint`, `clean` targets

---

[1.0.3]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v1.0.1...v1.0.2
[1.0.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.5.0...v1.0.0
[0.5.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/releases/tag/v0.1.0
