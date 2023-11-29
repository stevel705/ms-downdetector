# Downdetector server on Golang 

## Before run

Set environment variables:
```bash
export TELEGRAM_TOKEN=...
export CHAT_ID=...
```

or set it in environment variables in docker-compose.yml

Added vps_servers.json file with list of servers for check in json format
```json
"vps_name": [
        "https://",
        "http://",
        "http://"
    ],
```

## How to run

```bash
go run main.go
```

## How to build

```bash
go build main.go
```


## How to run docker compose

```bash
docker-compose up
```

## How to run docker compose in background

```bash
docker-compose up -d
```
