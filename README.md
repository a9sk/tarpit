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
   - To use HTTPS, set `tls-cert` and `tls-key`. Generate a self-signed `localhost` pair with:
     ```bash
     make certs
     ```
2. Run the server:
   ```bash
   ./tarpit
   ```
   To override the configured port temporarily, set the `PORT` environment variable:
   ```bash
   PORT=8080 ./tarpit
   ```
3. Open `http://localhost:<port>` (HTTP) or `https://localhost:<port>` (HTTPS) in a browser or use `curl`.
   - With the generated self-signed certificate, browsers will show a security warning; accept or trust `certs/localhost.crt` to suppress it.
   - On Linux, ports below 1024 (such as 80/443) require root or `CAP_NET_BIND_SERVICE`.

## Docker

```bash
docker build -t tarpit:latest .
docker run --tty --name tarpit -p "127.0.0.1:443:443" --rm tarpit:latest
```

Or with Docker Compose:

```bash
docker compose up
```

## Configuration

Configuration follows the same `config/config.json` format as Pyison:

- `port` — port to serve on.
- `tls-cert` / `tls-key` — paths to a TLS certificate and key; when both are set the server serves HTTPS.
- `random-seed` — global salt.
- `document-root` — path prefix (useful for reverse-proxy sub-paths).
- `fake-image-dir`, `fake-css-dir` — fake asset directory prefixes.
- `spacing-characters`, `unsafe-characters` — URL word separators and chars to strip from URLs.
- `robots-txt`, `html-templates`, `css-files`, `images` — asset paths (images may include `ico`, `jpg`, `png`, and `mp4`).
- `remove-from-stop-words` — stop words to exclude from generation.

## HTML Templating

The same template tags from Pyison are supported: `{HOME}`, `{TITLE}`, `{UPTITLE}`, `{MAIN}`, `{UP}`, `{CSSLINK}`, `{WORD}`, `{NAME}`, `{SENTENCE}`, `{PIC}`, `{LINK}`, `{OVER}`, `{NEWTITLE}`. `{PIC}` generates a fake asset path; append the desired extension such as `.jpg`, `.png`, or `.mp4`.

## Testing

```bash
go test ./...
```

## License

This project is licensed under the [European Union Public License v1.2](LICENSE) (EUPL-1.2).

It is a derivative of [Pyison](https://github.com/JonasLong/Pyison) by JonasLong, which was used under the MIT License. The original MIT copyright notice and permission notice are preserved in the `LICENSE` file.
