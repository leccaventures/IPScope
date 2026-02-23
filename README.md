# IPScope

IPScope is a Go metrics exporter that resolves each configured node IP into datacenter-like location metadata and exposes it through Prometheus metrics.

## Features

- Feature-based package layout (`config`, `geolocation`, `metrics`, `server`, `model`)
- YAML config (`config.yml`)
- Configurable bind address (`127.0.0.1` or `0.0.0.0`)
- Configurable port
- Custom Prometheus metric prefix
- Node IP to datacenter/location lookup (city, region, country, latitude, longitude)

## Config

Example `config.yml`:

```yaml
server:
    host: 127.0.0.1
    port: 9201

metrics:
    prefix: ipscope

nodes:
    - name: LECCA
      endpoint: 129.3.3.1
```

## Run

```bash
go run ./cmd/ipscope -config config.yml
```

## Docker Compose

Use the included compose setup:

```bash
docker compose up -d --build
```

Default compose files:

- `docker-compose.yml`
- `config.docker.yml` (binds to `0.0.0.0` for container networking)

Stop and remove:

```bash
docker compose down
```

Metrics endpoint:

- `http://127.0.0.1:9201/metrics` (when host is `127.0.0.1`)
- `http://0.0.0.0:9201/metrics` (when host is `0.0.0.0`)

## Exported Metrics

- `<prefix>_node_datacenter_info{node,endpoint,datacenter,city,region,country,latitude,longitude}`
- `<prefix>_node_datacenter_lookup_error{node,endpoint}` (1 = error, 0 = success)

## Notes

- Geolocation data is fetched from `http://ip-api.com/json/<ip>`.
- Free ip-api tier is rate-limited; the exporter surfaces `X-Rl` and `X-Ttl` in 429 error messages when provided.
- Datacenter label is derived primarily from `org` (fallback to `isp`).
