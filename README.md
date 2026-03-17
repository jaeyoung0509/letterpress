# letterpress

A terminal-first letter and card composer for beautiful printable messages.

## What ships today

`letterpress` is a Bubble Tea-based TUI and CLI for composing English-only letters and cards from YAML templates, text content, local images, and optional decorations.

Current capabilities:

- interactive TUI flow for template selection, page setup, text editing, image assignment, decoration toggles, review, and export
- starter templates for a classic letter and a modern card
- ISO page sizes `A3`, `A4`, `A5`, and `A6`
- non-interactive validation and render commands
- PDF and PNG export

## Requirements

- Go `1.25+`

## Getting started

Run the TUI directly from the repo:

```bash
go run .
```

You can also launch the explicit TUI subcommand:

```bash
go run . tui
```

The current TUI flow is:

1. Template Selection
2. Paper Size & Orientation
3. Content Composition
4. Image Assignment
5. Decoration Selection
6. Review & Export

## CLI commands

Inspect the available commands:

```bash
go run . --help
```

List shipped templates:

```bash
go run . templates list --dir templates
```

Validate a template and project pair:

```bash
go run . validate \
  --template templates/letter/classic-letter-a4.yaml \
  --project templates/samples/classic-letter-project.yaml
```

Render a composition without opening the TUI:

```bash
go run . render \
  --template templates/letter/classic-letter-a4.yaml \
  --project templates/samples/classic-letter-project.yaml \
  --format pdf \
  --out ./outputs/classic-letter.pdf
```

The `render` command accepts:

- `--template`: template YAML path
- `--project`: project YAML path
- `--format`: `pdf` or `png`
- `--out`: output path override

If `--out` is omitted, the command uses the project export target.

## Templates and samples

Shipped starter content lives under `templates/`:

- `templates/letter/classic-letter-a4.yaml`
- `templates/card/modern-card-a6.yaml`
- `templates/samples/classic-letter-project.yaml`
- `templates/samples/modern-card-project.yaml`

Assets used by the starter templates live under `templates/assets/`.

## Development

Local workflow helpers:

- `make run` launches the application entrypoint
- `make test` runs the full test suite
- `make fmt` formats the codebase with `go fmt`
- `make tidy` syncs module dependencies

Implementation work follows an issue -> branch -> PR workflow targeting `develop`.
