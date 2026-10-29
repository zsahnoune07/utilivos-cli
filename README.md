# utilivos-cli

`utilivos-cli` is the internal cross-project utility CLI for propagating shared constants across the Utilivos workspace.

Phase 1 in this module establishes the reusable CLI/runtime foundation.
Phase 2 adds the first production command: `theme sync` for cross-project theme color propagation.

## Scope

- Language/runtime: Go (`go run ./cmd/utilivos-cli ...`)
- Domains: theme colors; admin bundle publish (`publish`)
- Current targets:
  - `utilivos-mobile`
  - `utilivos-web`
  - `utilivos-sdk`
- Canonical export source: `utilivos-mobile`

## Commands

### publish

Upload a `.ubb` to the Utilivos admin API (`POST /v1/admin/plugins/publish`). The command reads `manifest.json` or `manifest` at the archive root, maps manifest `tier` to API `pricing_type` (`free` or `pro`), and sends `category_name` with the bundle. Tags, runtime metadata, and icon handling follow utilivos-api behavior.

**Environment file:** if `.env` exists in the current working directory, it is loaded first (keys already set in the process environment are not overwritten). Use `--env-file <path>` to point at a specific file.

**Primary environment variables:**

- `UTILIVOS_API_BASE_URL` — API origin (e.g. `http://localhost:8080`), no trailing slash required.
- `UTILIVOS_ADMIN_API_KEY` — value for `X-API-Key`.

**Legacy names** (used only when the primary variables are unset): `UTILIVO_API_URL`, `UTILIVO_ADMIN_API_KEY`.

Example `.env`:

```bash
UTILIVOS_API_BASE_URL=http://localhost:8080
UTILIVOS_ADMIN_API_KEY=your-admin-key
```

```bash
go run ./cmd/utilivos-cli publish ./dist/my-tool-1.0.0.ubb
```

**Interactive mode** (default when stdin is a TTY): loads admin categories, confirms or changes category (including creating a new slug/name), prompts for plugin status (`draft` / `published` / `archived`) and version status (`draft` / `published` / `deprecated`), optional changelog, then a final confirm before upload.

**Non-interactive** (`--non-interactive` or when stdin is not a terminal): no prompts. If the manifest category slug does not match any API category, pass `--category-slug` and, for a category that does not exist yet, `--category-name`. Use `--plugin-status` and `--version-status` to override the default `published` for both.

Flags:

- `--env-file`: Path to `.env` (optional; default tries `./.env`).
- `--url`: API base URL (overrides env).
- `--api-key`: Admin API key (overrides env).
- `--non-interactive`: Disable prompts (CI / scripts).
- `--category-slug`, `--category-name`: Resolve catalog mismatch without prompts.
- `--plugin-status`, `--version-status`: Lifecycle fields in non-interactive mode.
- `--changelog`: Optional changelog string.
- `--timeout`: HTTP client timeout (default 5m).
- `--json`: Print JSON (`message`, `plugin`, `version`, `category`).

The multipart `bundle` part uses a filename ending in `.ubb`, as required by the API.

### theme sync

`utilivos-cli theme sync`

Flags:

- `--config <path>`: theme config path. Default: `utilivos.theme.json` at workspace root.
- `--export`: force export mode.
- `--apply`: force apply mode.
- `--dry-run`: build full change plan without writing files.
- `--diff`: print value-level before/after role diffs.
- `--workspace <path>`: override workspace discovery start path.
- `--allow-missing`: continue when target projects are missing. Default is fail.

Global:

- `--verbose`: enables debug logging.

## Mode Resolution

If neither `--export` nor `--apply` is passed:

1. Config file exists: apply mode.
2. Config file does not exist: export mode.

Explicit `--export` or `--apply` always wins (except invalid `--export` + `--apply` together).

## Theme Config Schema (v1)

Config file: JSON only.

Top-level fields:

- `schemaVersion` (must be `1`)
- `seedColor` (`#RRGGBB`)
- `colors.light`
- `colors.dark`

Required semantic roles in both `colors.light` and `colors.dark`:

- `background`
- `surface`
- `primary`
- `onPrimary`
- `secondary`
- `onSecondary`
- `text`
- `muted`
- `error`
- `onError`
- `success`
- `warning`
- `surfaceAlt`
- `outline`

Validation rules:

- Unknown keys are rejected.
- Duplicate JSON keys are rejected.
- Missing required roles are rejected.
- Colors must be strict hex `#RRGGBB`.
- Errors are reported with actionable JSON paths.

Example:

```json
{
  "schemaVersion": 1,
  "seedColor": "#0F766E",
  "colors": {
    "light": {
      "background": "#F8FAFC",
      "surface": "#EEFCF9",
      "primary": "#00534D",
      "onPrimary": "#FFFFFF",
      "secondary": "#215055",
      "onSecondary": "#FFFFFF",
      "text": "#111E1C",
      "muted": "#394A48",
      "error": "#98000A",
      "onError": "#FFFFFF",
      "success": "#1A7F37",
      "warning": "#B05A00",
      "surfaceAlt": "#F1F5F9",
      "outline": "#576865"
    },
    "dark": {
      "background": "#0E1416",
      "surface": "#152025",
      "primary": "#00DECF",
      "onPrimary": "#002A26",
      "secondary": "#A1CED4",
      "onSecondary": "#00292D",
      "text": "#E2F1EE",
      "muted": "#B8CAC7",
      "error": "#FFB4AB",
      "onError": "#520003",
      "success": "#7EE787",
      "warning": "#FFB86B",
      "surfaceAlt": "#132126",
      "outline": "#8C9F9B"
    }
  }
}
```

## Export Behavior

`theme sync` export mode reads canonical values from:

- `utilivos-mobile/lib/core/theme/app_design_tokens.dart`

Then it writes the generated config file.

Mismatch strategy is locked:

- Export source remains mobile canonical.
- SDK/web values are compared against canonical.
- Mismatches are reported explicitly.
- Export still succeeds with warnings and deterministic config output.

## Apply Behavior

`theme sync` apply mode:

1. Loads + validates the config.
2. Builds deterministic per-project plans.
3. Applies only changed files (unless `--dry-run`).
4. Prints changed files and optional value-level diffs.

Update coverage:

- Mobile:
  - `utilivos-mobile/lib/core/theme/app_design_tokens.dart`
  - Updates `seedColor` and light/dark semantic palette values.
- Web:
  - `utilivos-web/app/globals.css`
  - Maintains a managed theme block with light/dark + media fallback.
  - Maintains mapping to legacy web vars (`--bg`, `--panel`, `--text`, `--muted`, `--primary`, `--border`).
- SDK:
  - `docs/design-system/tokens.json`
  - `utilivos-sdk/ui/base.css`
  - `utilivos-sdk/internal/templates/assets/base.css`
  - `utilivos-sdk/examples/*/tool/utilivos/base.css`
  - `utilivos-sdk/tests/fixtures/*/tool/utilivos/base.css`
  - Ensures SDK CSS color tokens are refreshed from updated semantics.

## Missing Targets

Default behavior:

- Missing required target files fail the command.

With `--allow-missing`:

- Missing targets are skipped.
- Command continues with warnings.

## Dry-Run and Diff

Dry-run only:

```bash
go run ./cmd/utilivos-cli theme sync --dry-run
```

Dry-run + role diffs:

```bash
go run ./cmd/utilivos-cli theme sync --dry-run --diff
```

`--diff` prints per-project semantic before/after values.

## End-to-End Workflow

1. Export current canonical config:

```bash
go run ./cmd/utilivos-cli theme sync
```

2. Edit `utilivos.theme.json`.

3. Preview changes:

```bash
go run ./cmd/utilivos-cli theme sync --dry-run --diff
```

4. Apply:

```bash
go run ./cmd/utilivos-cli theme sync
```

5. Re-run apply to verify idempotency (should report zero changed files when no config changes).

## Developer Notes

- Workspace discovery works from repo root or nested subdirectories.
- Optional settings file at workspace root: `.utilivos-cli.json`.
- Supported key:
  - `defaultThemeConfigPath`
- `internal/dotenv` implements minimal `.env` loading for commands such as `publish` (no third-party dotenv dependency).

## Tests

Run all tests:

```bash
GOCACHE=/tmp/go-cache go test ./...
```

Test coverage includes:

- Workspace discovery.
- Export from canonical mobile values.
- Config validation (invalid hex, unknown keys, duplicate keys).
- Apply flow across mobile/web/sdk.
- Mismatch reporting.
- Dry-run no-write behavior.
- Missing target behavior (default fail and `--allow-missing`).
- SDK CSS propagation.
- Idempotent second apply.
