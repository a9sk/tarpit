# Tarpit

A Go rewrite of [Pyison](https://github.com/JonasLong/Pyison), a tarpit for AI web crawlers.

## Overview

Tarpit serves an endless maze of dynamically generated, plausible-looking pages to trap AI web crawlers. Each page is deterministically generated from its URL and a global seed, so reloads return identical content.

The NLTK English word list and stop-word list are embedded into the binary, so there is no runtime dependency on Python or NLTK.

## Differences from Pyison

- Written in Go; no Python/NLTK installation required.
- The entire application lives in a single `tarpit.go` source file.
- Word lists are embedded with `//go:embed`.
- Uses Go's standard `net/http` server.
- Structured logging via `log/slog`.
- Deterministic randomness is provided by Go's `math/rand` seeded from the URL path and config salt.

## Building

```bash
go build .
```

## Running

1. Copy or edit `config/config.json`.
   - Set `random-seed` to a non-zero value.
   - Change `port` if needed.
2. Run the server:
   ```bash
   ./tarpit
   ```
   To override the configured port temporarily, set the `PORT` environment variable:
   ```bash
   PORT=8080 ./tarpit
   ```
3. Open `http://localhost:<port>` in a browser or use `curl`.

## Docker

```bash
docker build -t tarpit:latest .
docker run --tty --name tarpit -p "127.0.0.1:80:80" --rm tarpit:latest
```

Or with Docker Compose:

```bash
docker compose up
```

## Configuration

Configuration follows the same `config/config.json` format as Pyison:

- `port` — port to serve on.
- `random-seed` — global salt.
- `document-root` — path prefix (useful for reverse-proxy sub-paths).
- `fake-image-dir`, `fake-css-dir` — fake asset directory prefixes.
- `spacing-characters`, `unsafe-characters` — URL word separators and chars to strip from URLs.
- `robots-txt`, `html-templates`, `css-files`, `images` — asset paths.
- `remove-from-stop-words` — stop words to exclude from generation.

## HTML Templating

The same template tags from Pyison are supported: `{HOME}`, `{TITLE}`, `{UPTITLE}`, `{MAIN}`, `{UP}`, `{CSSLINK}`, `{WORD}`, `{NAME}`, `{SENTENCE}`, `{PIC}`, `{LINK}`, `{OVER}`, `{NEWTITLE}`.

## Testing

```bash
go test ./...
```

## License

This project is licensed under the [European Union Public License v1.2](LICENSE) (EUPL-1.2).

It is a derivative of [Pyison](https://github.com/JonasLong/Pyison) by JonasLong, which was used under the MIT License. The original MIT copyright notice and permission notice are preserved in the `LICENSE` file.
