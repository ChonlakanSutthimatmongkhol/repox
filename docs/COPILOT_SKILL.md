# Repox — Copilot Development Skill

## Project Overview

Repox is a CLI tool written in Go that scans a codebase, learns project conventions, and generates feature scaffolds matching the team's existing code style.

## Tech Stack

- **Language**: Go 1.26+
- **CLI Framework**: cobra (github.com/spf13/cobra)
- **Template Engine**: Go `text/template`
- **Git**: go-git (github.com/go-git/go-git/v5)
- **File Scanning**: `filepath.WalkDir`
- **JSON**: standard `encoding/json`
- **Testing**: standard `testing` + testify (github.com/stretchr/testify)

## Architecture Rules

1. All business logic lives in `internal/` — never in `cmd/`
2. Every major component has an interface defined in its own file
3. Config, conventions, examples, lessons, generations are stored as JSON in `.repox/` directory
4. Templates use Go `text/template` syntax with `.tmpl` extension
5. File writer must check for existing files and refuse to overwrite without `--force` flag
6. All commands return structured output to stdout
7. Use `snake_case` for generated file names, `PascalCase` for class names
8. Naming conversion utilities must handle: `watchlist` → `Watchlist`, `watch-list` → `WatchList`, `watch_list` → `WatchList`

## Directory Structure

```
repox/
├── cmd/repox/main.go
├── internal/
│   ├── cli/          # Cobra commands
│   ├── scanner/      # Repo scanning logic
│   ├── generator/    # Template & AI generation
│   ├── retriever/    # Example finding & similarity
│   ├── learner/      # Diff reading & lesson extraction
│   ├── ai/           # AI provider abstraction
│   ├── config/       # Config loading/saving
│   └── models/       # Shared data types
├── templates/        # Template files
├── go.mod
├── go.sum
└── Makefile
```

## Phase 1 Implementation Checklist

Build these in order. Each checkpoint must compile and pass tests before moving to the next.

### Checkpoint 1: Project Setup
- [ ] Initialize Go module: `github.com/ChonlakanSutthimatmongkhol/repox`
- [ ] Create directory structure as shown above
- [ ] Add cobra dependency
- [ ] Create `cmd/repox/main.go` with root command
- [ ] Create `internal/cli/root.go` — root command with version flag
- [ ] Verify: `go build ./cmd/repox` succeeds
- [ ] Verify: `./repox --version` prints version

### Checkpoint 2: Models & Config
- [ ] Create `internal/models/convention.go`:
  ```go
  type Convention struct {
      ProjectType      string            `json:"project_type"`
      StateManagement  string            `json:"state_management"`
      FeatureStructure string            `json:"feature_structure"`
      FeatureRoot      string            `json:"feature_root"`
      TestRoot         string            `json:"test_root"`
      Naming           NamingConvention  `json:"naming"`
      Routing          RoutingConfig     `json:"routing"`
      CommonImports    []string          `json:"common_imports"`
  }

  type NamingConvention struct {
      ClassCase       string `json:"class_case"`
      FileCase        string `json:"file_case"`
      ScreenSuffix    string `json:"screen_suffix"`
      BlocSuffix      string `json:"bloc_suffix"`
      EventSuffix     string `json:"event_suffix"`
      StateSuffix     string `json:"state_suffix"`
      RepositorySuffix string `json:"repository_suffix"`
      UsecaseSuffix   string `json:"usecase_suffix"`
  }

  type RoutingConfig struct {
      Type      string   `json:"type"`
      RouteFile string   `json:"route_file"`
  }
  ```
- [ ] Create `internal/models/config.go`:
  ```go
  type Config struct {
      Version         string    `json:"version"`
      ProjectType     string    `json:"project_type"`
      FeatureRoot     string    `json:"feature_root"`
      TestRoot        string    `json:"test_root"`
      DefaultTemplate string    `json:"default_template"`
      AI              AIConfig  `json:"ai"`
  }

  type AIConfig struct {
      Provider        string `json:"provider"`
      GenerationModel string `json:"generation_model"`
      LearningModel   string `json:"learning_model"`
  }
  ```
- [ ] Create `internal/models/generation.go`, `example.go`, `lesson.go`
- [ ] Create `internal/config/loader.go` — Load/Save JSON config files from `.repox/`
- [ ] Write tests for config loader
- [ ] Verify: tests pass

### Checkpoint 3: Naming Utilities
- [ ] Create `internal/generator/naming.go`:
  ```go
  // ToSnakeCase: "watchList" → "watch_list", "WatchList" → "watch_list"
  func ToSnakeCase(s string) string

  // ToPascalCase: "watchlist" → "Watchlist", "watch-list" → "WatchList", "watch_list" → "WatchList"
  func ToPascalCase(s string) string

  // ToCamelCase: "watchlist" → "watchlist", "watch-list" → "watchList"
  func ToCamelCase(s string) string
  ```
- [ ] Write comprehensive tests for all naming edge cases
- [ ] Verify: tests pass

### Checkpoint 4: Init Command
- [ ] Create `internal/cli/init.go`:
  - Creates `.repox/` directory
  - Creates default `config.json`
  - Creates empty `conventions.json`, `examples.json`, `lessons.json`, `generations.json`
  - Prints success message
  - Skips if `.repox/` already exists (with `--force` to override)
- [ ] Write tests
- [ ] Verify: `./repox init` creates `.repox/` with all files

### Checkpoint 5: Template Engine
- [ ] Create Flutter BLoC templates in `templates/flutter_bloc_feature/`:
  - `screen.dart.tmpl`
  - `bloc.dart.tmpl`
  - `event.dart.tmpl`
  - `state.dart.tmpl`
  - `repository.dart.tmpl`
  - `repository_impl.dart.tmpl`
  - `usecase.dart.tmpl`
  - `request.dart.tmpl`
  - `response.dart.tmpl`
  - `bloc_test.dart.tmpl`
- [ ] Create `internal/generator/template_generator.go`:
  - Load templates from embedded filesystem (`embed.FS`)
  - Render with feature name context
  - Return list of (path, content) pairs
- [ ] Create `internal/generator/file_writer.go`:
  - Write files to disk
  - Create directories as needed
  - Check for existing files — refuse without `--force`
  - Return summary of written files
- [ ] Write tests
- [ ] Verify: template rendering produces valid Dart code

### Checkpoint 6: Generate Command
- [ ] Create `internal/cli/generate.go`:
  - Subcommand: `repox generate feature <name>`
  - Load config from `.repox/config.json`
  - Load conventions from `.repox/conventions.json`
  - Render templates
  - Write files
  - Print summary tree
  - Flags: `--force`, `--dry-run`, `--template`
- [ ] Write integration tests
- [ ] Verify: `./repox generate feature watchlist` creates correct file tree
- [ ] Verify: `./repox generate feature watchlist --dry-run` shows preview without writing
- [ ] Verify: running again without `--force` refuses to overwrite

### Checkpoint 7: Makefile & Polish
- [ ] Create Makefile:
  ```makefile
  build:
      go build -o bin/repox ./cmd/repox

  test:
      go test ./... -v

  install:
      go install ./cmd/repox

  lint:
      golangci-lint run

  clean:
      rm -rf bin/
  ```
- [ ] Embed templates using `//go:embed`
- [ ] Add `--version` flag
- [ ] Add colorized output (fatih/color or similar)
- [ ] Verify: full flow works end-to-end

## Template Context

Every template receives this context:

```go
type TemplateContext struct {
    FeatureName     string // "watchlist"
    PascalName      string // "Watchlist"
    CamelName       string // "watchlist"
    SnakeName       string // "watchlist"
    ScreenSuffix    string // "Screen"
    BlocSuffix      string // "Bloc"
    EventSuffix     string // "Event"
    StateSuffix     string // "State"
    RepositorySuffix string // "Repository"
    UsecaseSuffix   string // "UseCase"
    CommonImports   []string
}
```

## Template Example: `bloc.dart.tmpl`

```dart
import 'package:flutter_bloc/flutter_bloc.dart';

part '{{.SnakeName}}_event.dart';
part '{{.SnakeName}}_state.dart';

class {{.PascalName}}{{.BlocSuffix}} extends Bloc<{{.PascalName}}{{.EventSuffix}}, {{.PascalName}}{{.StateSuffix}}> {
  {{.PascalName}}{{.BlocSuffix}}() : super(const {{.PascalName}}Initial()) {
    on<{{.PascalName}}Started>(_onStarted);
  }

  Future<void> _onStarted(
    {{.PascalName}}Started event,
    Emitter<{{.PascalName}}{{.StateSuffix}}> emit,
  ) async {
    emit(const {{.PascalName}}Loading());
    // TODO: implement
    emit(const {{.PascalName}}Loaded());
  }
}
```

## Error Handling Rules

- If `.repox/` doesn't exist and command needs it → print: "Run `repox init` first"
- If config file is invalid JSON → print error with file path and exit
- If template is missing → print which template and exit
- If target file already exists → print "File exists: <path>. Use --force to overwrite"
- Never panic — always return meaningful error messages

## Code Style

- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Use `slog` for structured logging
- Keep functions under 50 lines where possible
- Every exported function has a doc comment
- Group imports: stdlib, external, internal

## Testing Strategy

- Unit tests for: naming utilities, config loader, template rendering
- Integration tests for: init command (temp dir), generate command (temp dir)
- Test file names: `*_test.go` in same package
- Use `t.TempDir()` for filesystem tests
- Use testify `assert` and `require`

## Future Phases (DO NOT implement yet, structure only)

Phase 2 will add `repox scan` — scanner implementations go in `internal/scanner/`
Phase 3 will add example retrieval — retriever implementations go in `internal/retriever/`
Phase 4 will add AI generation — AI client goes in `internal/ai/`
Phase 5 will add learning — learner implementations go in `internal/learner/`
