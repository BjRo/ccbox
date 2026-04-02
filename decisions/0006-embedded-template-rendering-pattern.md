# ADR-0006: Embedded template rendering pattern

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-dttd

## Context

The `render` package needs to produce multiple output files (shell scripts, config files, Dockerfiles, devcontainer.json) from Go templates parameterized by `GenerationConfig`. The firewall script templates (ccbox-dttd) were the first templates added to the codebase, establishing a pattern that subsequent template beans (ccbox-v1zh Dockerfile, ccbox-v9jt devcontainer.json, ccbox-780o mise.toml, ccbox-7qvl Claude settings) must follow.

Key decisions were: where templates live on disk, how they are parsed, how custom template functions are registered, how rendering functions are structured, and how to defend against injection when templates produce shell scripts.

## Decision

### Template file layout

Templates live in `internal/render/templates/` as `.tmpl` files, embedded into the binary via `//go:embed` and `embed.FS`. This keeps templates separate from Go source while bundling them at compile time with zero runtime file I/O.

### Parsing

Templates are parsed once at package level using `template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.tmpl"))`. This fails fast at program startup if any template has a syntax error, which is correct for embedded templates that should always be valid. The `Funcs(funcMap)` call must come before `ParseFS` because Go's template parser needs to know about custom functions at parse time.

### FuncMap helpers

Custom template functions are registered via a package-level `template.FuncMap`. Each helper should be minimal (a single `strings.TrimPrefix` call, not a validation+transformation pipeline). Input validation happens upstream in `render.Merge` / `firewall.Merge`, not in template helpers.

### Pure rendering functions

Each rendering function (e.g., `RenderFirewall(cfg GenerationConfig) (FirewallFiles, error)`) is a pure transformation: `GenerationConfig` in, rendered bytes out, no file I/O. Actual file writing is deferred to the orchestrator (`ccbox init` command). This keeps rendering testable without touching the filesystem.

Return types use `[]byte` (not `string`) because downstream `os.WriteFile` wants bytes.

### Shell injection defense (two-layer)

Templates that produce shell scripts use two independent defense layers:

1. **Input validation**: `firewall.ValidateDomain` rejects any domain name that does not match strict RFC 1123 DNS hostname syntax. This runs in `firewall.Merge` before any domain enters `GenerationConfig`. Shell metacharacters (`;`, `$`, `` ` ``, `|`, etc.) are structurally impossible in valid DNS names.
2. **Output quoting**: All domain name interpolations in shell templates use single quotes (e.g., `dig +short '{{.Name}}'`). Single quotes prevent shell expansion even if validation were bypassed.

Both layers are independently sufficient. Together they provide defense-in-depth.

### text/template, not html/template

Generated outputs are shell scripts and config files, not HTML. `html/template` would HTML-escape characters that are meaningful in shell context (e.g., `&` to `&amp;`). `text/template` performs no escaping, which is correct when paired with the two-layer injection defense above.

## Consequences

- All future template beans add `.tmpl` files to `internal/render/templates/` and rendering functions to `internal/render/`.
- FuncMap helpers are shared across all templates in the package. Name them descriptively to avoid collisions as the set grows.
- Templates that generate non-shell output (Dockerfile, JSON, TOML) do not need the single-quote defense layer but still benefit from upstream input validation.
- Static templates (no Go template variables, like `warmup-dns.sh`) are still `.tmpl` files rendered through `text/template` for pipeline uniformity.
- Testing follows the structural-assertion pattern (contains expected strings, starts with expected prefix) rather than golden-file snapshots. See the "Testing Template Output" section in CLAUDE.md.
