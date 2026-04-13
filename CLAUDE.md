# secmsg

## Project Status: UNRELEASED

This is an unreleased project. Do not retain legacy code or implement backward compatibility. Remove deprecated code rather than keeping it.

## About

`secmsg` is an MIT-licensed IM client library and CLI for communicating with `sigd` over JSON-RPC 2.0.

## Package Layout

- `schema/` — shared notification types (method constants, params structs)
- `client/` — client library (connect, send, subscribe, RPC methods)
- `cmd/secmsg/` — CLI reference implementation

## Building

```bash
go build ./...
go test ./...
```

## Default Connection

`127.0.0.1:9801` (sigd default port)
