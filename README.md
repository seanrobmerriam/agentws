# agentws

`agentws` is a small Go CLI that scaffolds an **agent workspace** in the
current working directory by merging resources from a source configuration
directory.

## Source layout

By default `agentws` reads from `$AGENTWS_HOME` if set, otherwise
`~/.config/agentws/`. Override with `--source DIR`.

The source directory may contain any subset of:

| Path                | Purpose                                              |
|---------------------|------------------------------------------------------|
| `skills/`           | Agent skills (e.g. `skills/example/SKILL.md`)        |
| `instructions/`     | Agent system prompts                                 |
| `agents/`           | `AGENTS.md` files and similar agent-level guidance   |
| `docs/`             | General documents to include in the project          |
| `agentws.toml`      | Project configuration file (copied to target root)   |

Missing subdirectories are treated as optional and silently skipped.

## Usage

```sh
# Build
go build -o agentws ./cmd/agentws

# Scaffold into the current directory (merge)
./agentws create

# Overwrite conflicting files
./agentws create --force

# Use a different source
./agentws create --source /path/to/source

# Override the default source via env
AGENTWS_HOME=/path/to/source ./agentws create
```

## Merge semantics

`create` walks each known subdirectory in the source and copies files into
the matching subdirectory of the current working directory. Files that
already exist in the target are **preserved** unless `--force` is supplied,
in which case they are removed and replaced with the source content.

## Exit codes

| Code | Meaning                                            |
|------|----------------------------------------------------|
| 0    | Success (or all conflicts were skipped)            |
| 1    | Source missing, not a directory, or I/O error      |
| 2    | Invalid usage                                      |

## Project layout

```
.
├── cmd/agentws/        # CLI entry point
├── internal/create/    # Create implementation + tests
├── testdata/           # Test fixture config (used by tests only)
└── go.mod
```

## Testing

```sh
go build ./...
go test ./... -count=1
```

The test suite never reads or writes `~/.config/agentws/`; it uses
`testdata/fixture-config/`.
