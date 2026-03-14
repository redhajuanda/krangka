## CLI Tool (`cli/`)

The CLI tool for generating new Krangka projects.

```
cli/
└── krangka/
    ├── generator/                # Code generation logic
    │   └── generator.go
    ├── gonew/                    # Project template
    │   ├── examples/             # Example templates
    │   ├── exclude-config.json   # File exclusion rules
    │   ├── gonew.go              # Template generation
    │   └── README.md
    ├── utils/                    # CLI utilities
    │   └── stringx/              # String utilities
    ├── go.mod                    # CLI module definition
    ├── go.sum                    # CLI module checksums
    ├── Makefile                  # CLI build commands
    └── root.go                   # CLI root command

# Krangka Gonew Command

The `gonew` command generates new projects from the Krangka boilerplate with built-in exclusion capabilities.

## Usage

```bash
krangka new <package-name> [project-name] [options]
```

### Examples

```bash
# Basic usage
krangka new github.com/mycompany/myapp myapp

# With custom source module
krangka new github.com/mycompany/myapp myapp -m github.com/mycompany/krangka
```

## Options

- `-m, --source-mod`: Source module to use as boilerplate (default: github.com/mycompany/krangka)

## Exclusion Configuration

The exclusion configuration is embedded into the CLI binary using the `exclude-config.json` file. This ensures the configuration is always available when the CLI is distributed and used.

### Configuration File Format

The embedded `exclude-config.json` file has the following structure:

```json
{
  "directories": [
    "cli",
    ".git",
    ".vscode",
    ".idea",
    "node_modules",
    "vendor",
    "docs",
    "scripts"
  ],
  "files": [
    "*.log",
    "*.tmp",
    ".env*",
    "Dockerfile",
    "docker-compose.yml",
    "Makefile",
    "README.md",
    "go.sum",
    ".gitignore",
    ".gitattributes",
    "*.swp",
    "*.swo",
    "*~",
    ".DS_Store",
    "Thumbs.db"
  ],
  "code_blocks": [
    "exclude-start",
    "exclude-end"
  ]
}
```

### Directory Exclusions

Directories specified in the `directories` array will be completely skipped during project generation. The default exclusions include:
- `cli` - CLI tool itself
- `.git` - Git repository data
- `.vscode`, `.idea` - IDE configuration files
- `node_modules`, `vendor` - Dependencies
- `docs`, `scripts` - Documentation and scripts

### File Exclusions

Files matching patterns in the `files` array will be excluded. Supports glob patterns:
- `*.log` - Excludes all .log files
- `*.tmp` - Excludes all .tmp files
- `.env*` - Excludes all .env files
- `Dockerfile`, `docker-compose.yml` - Docker files
- `Makefile`, `README.md` - Build and documentation files
- `go.sum` - Go module checksums
- `.gitignore`, `.gitattributes` - Git configuration files
- `*.swp`, `*.swo`, `*~` - Editor temporary files
- `.DS_Store`, `Thumbs.db` - OS-specific files

### Code Block Exclusions

You can exclude specific code blocks from files using special comments:

```go
// This code will be included
func includedFunction() {
    // ...
}

// exclude-start
// This entire block will be excluded
func excludedFunction() {
    // This function won't be in the generated project
}
// exclude-end

// This code will be included again
func anotherIncludedFunction() {
    // ...
}
```

## Examples

### Example 1: Basic Project Generation

```bash
krangka new github.com/mycompany/myapp myapp
```

This will:
- Generate a new project in the `myapp` directory
- Use the default Krangka boilerplate
- Apply all exclusions defined in `exclude-config.json`
- Update all import paths and package names

### Example 2: Custom Source Module

```bash
krangka new github.com/mycompany/myapp myapp -m github.com/mycompany/krangka
```

This uses a custom source module while still applying the same exclusions.

## Code Block Exclusion Syntax

To exclude specific code blocks, wrap them with special comments:

```go
// exclude-start
// This entire block will be removed from the generated project
type ExcludedStruct struct {
    Field1 string
    Field2 int
}

func excludedFunction() {
    // This function will not be included
}
// exclude-end

// This code will be included
type IncludedStruct struct {
    Field1 string
    Field2 int
}
```

## Customizing Exclusions

To customize what gets excluded:

1. **Modify the embedded configuration**: Edit the `exclude-config.json` file in the CLI source directory and rebuild
2. **Create a local configuration**: Place an `exclude-config.json` file in your current working directory when running the command

The CLI will first look for `exclude-config.json` in the current directory (for local customization), then fall back to the embedded configuration.

## Notes

- If a directory is excluded, all its subdirectories and files are also excluded
- File patterns support glob syntax
- Code block exclusions work on a line-by-line basis
- The default exclusion includes the `cli` directory to avoid including the CLI tool itself in generated projects
- You can have multiple exclusion blocks in the same file 