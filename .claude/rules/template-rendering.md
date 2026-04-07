---
description: Template rendering conventions — embed pattern, rendering signatures, shell injection defense, Dockerfile whitespace
globs: "internal/render/**"
---

# Template Rendering Patterns

## Embedded Template Rendering

All templates live in `internal/render/templates/*.tmpl` and are embedded via `//go:embed` + `embed.FS`. See ADR-0006.

1. **Shared `embed.go`**: Single `//go:embed templates/*` directive exports `templateFS`. Individual render files do not declare their own embeds.
2. **Package-level `template.Must(template.ParseFS(...))`**: Parsed once at init. Parse failures surface immediately at startup.
3. **FuncMap before ParseFS**: `template.New("").Funcs(funcMap).ParseFS(...)` -- the parser needs to know custom functions at parse time.
4. **FuncMap helpers are minimal**: Single transformations only. Validation happens upstream in `Merge` functions.
5. **Pure rendering functions**: `RenderXxx(cfg GenerationConfig) (XxxFiles, error)` -- config in, bytes out, no file I/O.
6. **Uniform render signature**: `FuncName(w io.Writer, cfg GenerationConfig) error` for single-template renders.
7. **Use `text/template`, not `html/template`**: Output is config files, shell scripts, JSON -- not HTML.

## Non-nil Empty Slices for Templates

Functions that produce slices consumed by Go templates must return `[]T{}` instead of `nil` when empty. This avoids `nil` vs empty confusion in `{{range}}` and `{{if}}` template actions.

## Dockerfile Whitespace Continuations

Dockerfile `RUN` blocks use backslash continuation. When `{{ range }}` appends items, place it inline on the last static line:

```
    build-essential jq fzf{{ range .SystemDeps }} \
    {{ . }}{{ end }} \
    && rm -rf /var/lib/apt/lists/*
```

Never use `{{- }}` trim markers that would collapse continuation backslashes.

## JSON Safety in Templates

Templates producing JSON files need two safeguards:

1. **`jsonString` FuncMap helper**: Use `json.Marshal` on the string, then strip the surrounding quotes. The result is safe for interpolation inside JSON double quotes.
2. **Comma-separated arrays via indexed range**: JSON forbids trailing commas. Use `{{range $i, $v := .Items}}{{if $i}}, {{end}}"{{$v | jsonString}}"{{end}}`.

## Markdown Template Whitespace

Markdown templates need different whitespace handling than Dockerfile templates. Use `{{- }}` trim markers on `{{ end }}` tags around conditional blocks to prevent double blank lines. This is the opposite guidance from Dockerfile templates, where `{{- }}` must be avoided to preserve backslash continuations.

## Shell Script Temp File Cleanup

Generated shell scripts that create temp files use the `trap`/`EXIT` pattern:

```bash
TMPFILE="$(mktemp)"
trap 'rm -f "$TMPFILE"' EXIT
# ... use $TMPFILE ...
mv "$TMPFILE" "$DESTINATION"
trap - EXIT
```

Clear the trap after successful `mv` so the moved file is not deleted.

## Shell Injection Defense

Templates producing shell scripts use two independent defense layers:

1. **Input validation**: `firewall.ValidateDomain` enforces strict RFC 1123 DNS hostname syntax. Shell metacharacters are structurally impossible in valid DNS names.
2. **Output quoting**: All user-influenced interpolations use single quotes (e.g., `dig +short '{{.Name}}'`).

Both layers are independently sufficient.

## Heredoc Quoting Split for Shell Config Templates

When a shell script template writes a config file that needs both shell variable expansion and Go template-rendered content, split the write into two parts:

1. **`echo` statements** for lines needing shell variable expansion (e.g., `server=${UPSTREAM_DNS}`)
2. **Quoted heredoc** (`<< 'EOF'`) for Go template-rendered content (e.g., ipset directives)

This preserves the output quoting defense layer -- bash does not interpret `$`, backticks, or other metacharacters inside quoted heredocs, keeping template-rendered domain names safe even if they somehow bypassed input validation.

## Script Operation Ordering

Network-dependent operations in generated shell scripts (DNS resolution, CIDR fetches via curl) must precede `iptables -P OUTPUT DROP`. Once the default DROP policy is applied, outbound connections to unallowed IPs will fail. Add comments in templates noting this ordering dependency to prevent future refactors from accidentally reordering.

For scripts that run on every container start (via `postStartCommand`), use idempotent state caching (e.g., caching upstream DNS to a file) so restarts work correctly even when `/etc/resolv.conf` has been rewritten to point at a local resolver.

## Node Always Included

Node/npm is always present in generated containers (Claude Code requires it). The `render.EnsureNode` helper guarantees node is in `GenerationConfig.Runtimes` before any template renders. Templates iterate all runtimes uniformly with no node special-casing. See ADR-0008.

## Standalone Config Files Over Inline Heredocs

When a generated file (e.g., mise `config.toml`) should be user-editable after generation, extract it to a standalone template and `COPY` it in the Dockerfile rather than generating it inline with a COPY heredoc. This makes the file the single source of truth for its content (e.g., runtime versions) and allows users to edit it without touching the Dockerfile.
