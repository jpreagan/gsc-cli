# gsc-cli

A command line tool for the Google Search Console API.

## Install

```bash
go install github.com/jpreagan/gsc-cli@latest
```

## Quick Start

### 1) Create OAuth credentials (Google Cloud Console)

1. Enable the **Google Search Console API** in the same Google Cloud project that owns the OAuth client JSON you will download.
2. Create an OAuth client ID: **Desktop app** (installed app).
3. Configure the OAuth consent screen.
   - In Google Cloud, the consent screen has a "Publishing status". For personal use you can leave it as **Testing**, but you must add your Google account as a **test user** or Google will block the OAuth approval.
   - Consent branding (app name/logo) is project-level and shared by all OAuth clients in that project.

### 2) Store client credentials

Download the OAuth client JSON from Google Cloud Console, then:

```bash
gsc auth credentials ~/Downloads/client_secret_....json
```

This writes:

- `~/.gsc/credentials/<account>/client.json`

Default account name is `default` (override with `--account <name>`).
If you use multiple Google identities, keep them in separate local account slots (for example `--account personal` and `--account workspace`).

### 3) Authorize (store OAuth token)

Default (recommended): opens a browser and completes automatically via loopback callback:

```bash
gsc auth add
```

In the browser, choose the Google account that has access to your Search Console property.

Optional: read-only scope:

```bash
gsc auth add --readonly
```

If the browser cannot be opened (headless/SSH): manual copy/paste flow:

```bash
gsc auth add --manual
```

This prints an auth URL. Open it in a browser, approve access, then copy the full redirect URL from the browser address bar and paste it back into the terminal (the browser page may fail to load; that's ok).

Tokens are stored at:

- `~/.gsc/credentials/<account>/token.json`

### 4) Run commands

List sites (properties) visible to the authorized account:

```bash
gsc sites
```

If this is empty, your token is valid but that Google account has no visible Search Console properties. Re-run `gsc auth add` with the correct account, or grant that account access in Search Console.

Query Search Analytics:

```bash
gsc analytics --site sc-domain:example.com --dimensions query,page --row-limit 50
```

If `--start-date`/`--end-date` are omitted, `gsc analytics` defaults to the last 28 days ending 2 days ago (in `--tz`).

Inspect a URL:

```bash
gsc inspect --site sc-domain:example.com --url https://example.com/some/page
```

Manage sitemaps:

```bash
gsc sitemaps --site sc-domain:example.com
gsc sitemaps submit --site sc-domain:example.com https://example.com/sitemap.xml
```

## Site URL Formats

Search Console APIs use the "site URL" identifier:

- Domain property: `sc-domain:example.com`
- URL-prefix property: `https://example.com/`

## Output Modes

- Default: table
- `--plain`: tab-separated (no headers)
- `--json`: stable JSON for scripting

## Accounts

`gsc` stores credentials per named account (not necessarily the Google email):

```bash
gsc --account default auth list
gsc --account default auth credentials ~/Downloads/client_secret_....json
gsc --account default auth add
```

Account names must be `a-zA-Z0-9` plus `_` or `-` (max 64 chars).

## Development

```bash
gofmt -w .
go vet ./...
go test ./...
go run . --help
```
