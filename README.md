# exporter-care

A helper that automatically register exporter to consul during startup and deregister service when receive signals.

## Build

```bash
EGO_ENABLED=0 GOOS=linux go build -o exporter-care main.go
```

## Run

```bash
exporter-care path_to_exporter_or_exporter_name --consul-cluster http://x.x.x.x:8500
```
