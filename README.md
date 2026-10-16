# mdocs

Documentation from Markdown files, built for people and for AI agents. A single binary turns your `.md` files into:

- **A website** with your company's identity (colors taken from your logo).
- **A navigable PDF**: cover, table of contents with links and page numbers, bookmarks, a "page / total" footer, and links between documents that work inside the PDF.
- **A `.drawio` file** with Mermaid diagrams as **editable shapes** (flowchart, sequence, state, class). Other diagram types are exported as images.

It works offline: Mermaid and the templates are embedded in the binary.

## Use mdocs as a skill (Claude and other agents)

The recommended way to use mdocs is to ask an agent: *"document this project"*, *"add an installation page and build the PDF"*, *"export the diagrams to draw.io"*. The skill [`skills/mdocs/SKILL.md`](skills/mdocs/SKILL.md) follows the [Agent Skills](https://code.claude.com/docs/en/skills) format. It tells the agent how to:
- find the binary,
- understand the project (`context`, `check`),
- create and delete pages,
- follow the Markdown rules,
- produce the PDF, the draw.io file or the static site.

All of those commands have `--json` output.

**Claude Code**: install the binary (see *Installation*), then:

```bash
mdocs skill install --global          # ~/.claude/skills/mdocs/  → all your projects
mdocs skill install                   # .claude/skills/mdocs/    → this repository only (commit it for your team)
```

Claude loads it automatically when the task is about documentation, or you can call it with `/mdocs`. With `--with-binary`, the executable is copied into the skill too, in case `mdocs` is not on your `PATH`.

**claude.ai** (Pro, Max, Team and Enterprise, with code execution enabled):
1. Download `mdocs-skill_<version>.zip` from the [latest release](https://github.com/cristianino/md-docs-viewer/releases). It already contains `SKILL.md` and the Linux binary, which is what Claude's code execution environment runs.
2. Upload it under *Settings → Capabilities → Skills → Upload skill*.

You can also build the zip yourself with `mdocs skill zip --with-binary`, run from a linux/amd64 binary. Chrome is usually not available there, so stick to `check`, `context`, `new`/`delete` and `export --no-pdf --no-drawio`.

**Other agents** (Codex, Cursor, Copilot…): `mdocs init` creates an `AGENTS.md` with the same rules, and `mdocs agents` regenerates it. `mdocs skill print` prints the full skill so you can paste it into another tool's instructions.

## Installation

**From a release (no Go needed):** download the package for your platform from [Releases](https://github.com/cristianino/md-docs-viewer/releases): `mdocs_<version>_linux_amd64.tar.gz`, `…_windows_amd64.zip`, `…_darwin_arm64.tar.gz`, and so on. Each package contains the binary, `brand.config.json`, `docs.config.json`, sample `.md` files, `logo.svg`, `AGENTS.md`, `skill/` and `deploy/`. No source code is included. The files are built once, when the release is published, and `checksums.txt` lets you verify them.

```bash
tar xzf mdocs_1.0.0_linux_amd64.tar.gz && cd mdocs_1.0.0_linux_amd64
./mdocs serve                  # http://localhost:3000
```

**With Docker:** `docker run -p 8080:8080 -v "$PWD:/work:ro" ghcr.io/cristianino/mdocs` (or `:slim`, without Chrome).

**From source:** see *Quick start* below.

## Build requirements

- [Go 1.26+](https://go.dev/dl/) to build.
- Chrome or Chromium, only to generate the PDF and the `.drawio` file. If it is not detected, use `--chrome <path>` or the `MDOCS_CHROME` variable.

## Quick start

```bash
go build -o mdocs .            # on Windows: go build -o mdocs.exe .

mdocs init my-portal           # asks for company, title, description, year and "powered by" (off by default)
cd my-portal
mdocs new "User guide"         # creates docs/02-user-guide.md and registers it
mdocs serve                    # http://localhost:3000
mdocs build                    # dist/*.pdf and dist/*.drawio
```

Without prompts (scripts and AI):

```bash
mdocs init --yes --company "Acme" --title "Technical manual" --description "Internal API" --year 2026
```

## Commands

| Command | Purpose |
|---|---|
| `mdocs init [dir]` | Creates the configuration, `logo.svg`, a sample document and `AGENTS.md` |
| `mdocs serve [-p 3000] [--host] [--base-path] [--exports live\|static\|off]` | Web viewer; re-reads `.md` files and templates on every request |
| `mdocs build [pdf\|drawio\|all] [-o dist] [--mode auto\|image]` | Generates the files without running `serve` |
| `mdocs export [-o site] [--base-path /docs]` | Static site with the PDF and `.drawio` included (S3, Pages, Netlify, nginx) |
| `mdocs new "<title>"` | Creates `docs/NN-slug.md` and registers it in the index |
| `mdocs delete <id> [-y]` | Removes it from the index and deletes the `.md` (warns about documents that link to it) |
| `mdocs add <file.md>` | Registers an existing `.md` (and creates it if missing) |
| `mdocs index [--prune]` | Registers every loose `.md` in `docs/`; `--prune` removes entries whose file is gone |
| `mdocs remove <id>` | Removes it from the index without deleting the file |
| `mdocs list` | Lists the registered documents |
| `mdocs check [--strict]` | Validates files, ids, links, anchors and images (exit 1 on errors) |
| `mdocs context` | Summarizes the project (headings, anchors, diagrams, issues) |
| `mdocs agents` | Writes `AGENTS.md` with the rules for AI agents |
| `mdocs skill install [--global]` · `skill zip` · `skill print` | Installs the skill for Claude Code, builds the `.zip` for claude.ai, or prints it |
| `mdocs generate-styles` | Takes the colors from `logo.svg` and saves them to `brand.config.json` |
| `mdocs healthcheck` | Queries `/healthz` (for Docker's HEALTHCHECK) |

Global flags: `-C <dir>` to work on another folder, and `--chrome <path>` to point to Chrome. `new`, `delete`, `index`, `list`, `check`, `context`, `build` and `export` accept `--json`.

## Navigation

- **General index** at `/indice`, with a filter: every document and its sections.
- **"On this page"**: a button at the top right (key `t`) that starts collapsed and highlights the visible section. On wide screens the content narrows instead of being covered.
- **`[TOC]`**: on a line of its own inside a `.md`, inserts that document's table of contents, both on the web and in the PDF.

## Deployment

| Where | How |
|---|---|
| S3 + CloudFront, GitHub Pages, Netlify, nginx | `mdocs export --base-path /path`: static HTML; Chrome is only needed while exporting ([Pages workflow](deploy/github-pages.yml)) |
| VPS | `mdocs serve --host 127.0.0.1 --base-path /docs` behind nginx ([nginx](deploy/nginx.conf), [systemd](deploy/mdocs.service)) |
| Docker | `ghcr.io/cristianino/mdocs` (with Chromium) or `:slim` (no Chrome, `--exports static`); build locally with `docker build [--target slim] .` ([compose](deploy/docker-compose.yml)) |
| AWS Lambda | `slim` image + Lambda Web Adapter, with the PDF and `.drawio` built in CI ([Dockerfile](deploy/lambda/Dockerfile)) |

`serve` variables: `MDOCS_HOST`, `PORT`, `MDOCS_BASE_PATH`, `MDOCS_EXPORTS`, `MDOCS_DIST`, `MDOCS_CHROME`, `MDOCS_NO_SANDBOX`. See the wiki's *Deployment* page for details.

## Configuration

`brand.config.json` (created by `mdocs init`):

```json
{
  "primaryColor": "#1a2035",
  "accentColor": "#0066cc",
  "companyName": "My Company",
  "docsTitle": "Documentation Portal",
  "description": "Short description for the PDF cover",
  "docsYear": "2026",
  "poweredBy": false
}
```

`docs.config.json` is the ordered list of documents (`id`, `file`, `title`, `subtitle`). It is maintained by `new`, `delete`, `add`, `index` and `remove`.

## More information

The [wiki](https://github.com/cristianino/md-docs-viewer/wiki) explains every command, the PDF and draw.io exports, the templates, deployment and the AI agent guide.

## Contributing

PRs to `master` must pass CI: formatting, `go vet`, tests on Linux, Windows and macOS, an end-to-end test, the Docker images and the packaging. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).
