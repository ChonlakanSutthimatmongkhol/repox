# Changelog

All notable changes to Repox are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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

[0.2.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ChonlakanSutthimatmongkhol/repox/releases/tag/v0.1.0
