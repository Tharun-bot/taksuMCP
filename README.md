# taksuMCP

**taksuMCP** makes any MCP (Model Context Protocol) server's task handling
spec-compliant — correct state machine, TTL/expiry, idempotent task IDs,
and credential scoping under stateless auth — with pluggable storage
backends (SQLite by default, Postgres for scale).

It targets the MCP **Tasks extension** (SEP-2133), which ships as an
independently-versioned extension in the 2026-07-28 MCP spec release,
not as a core primitive.

> Not a job queue. Not a Temporal competitor. A focused, spec-conformant
> implementation of one part of the MCP spec, so you don't have to
> hand-roll it.

## Why

As of mid-2026, no off-the-shelf MCP client or server toolkit ships a
conformant implementation of the Tasks extension. taksuMCP is a thin,
dependency-light Go library plus a self-hostable reference server and
HTMX dashboard so you can see task state live.

## Quickstart (60 seconds)

```bash
git clone https://github.com/Tharun-bot/taksuMCP.git
cd taksuMCP
go run ./cmd/server
```

Open http://localhost:8080 to see the dashboard.

Or with Docker:

```bash
docker compose up
```

## Status

🚧 Early development. See [CHANGELOG.md](./CHANGELOG.md) and open
[issues](https://github.com/Tharun-bot/taksuMCP/issues) for current
progress. Not yet recommended for production use.

## Project layout