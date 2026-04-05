# Stockyard Tournament

**Self-hosted tournament brackets and event management**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted tools.

## Quick Start

```bash
curl -fsSL https://stockyard.dev/tools/tournament/install.sh | sh
```

Or with Docker:

```bash
docker run -p 9804:9804 -v tournament_data:/data ghcr.io/stockyard-dev/stockyard-tournament
```

Open `http://localhost:9804` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9804` | HTTP port |
| `DATA_DIR` | `./tournament-data` | SQLite database directory |
| `STOCKYARD_LICENSE_KEY` | *(empty)* | License key for unlimited use |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 5 records | Unlimited |
| Price | Free | Included in bundle or $29.99/mo individual |

Get a license at [stockyard.dev](https://stockyard.dev).

## License

Apache 2.0
