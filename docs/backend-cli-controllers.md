# CLI Controllers

Command-line controllers live in `cmd/wade/internal/controllers` and own argument handling, process output and lifecycle presentation. The CLI router dispatches commands through a common `HandleArgs` interface and injects the shared `settings` Model where required.

WADE's CLI controller set includes:

- [`api`](#api)
- [`config`](#config)
- [`help`](#help)
- [`lifecycle`](#lifecycle)

## API

The `api` controller exposes WADE's HTTP API operations as commands for
scripts and coding agents. It parses the embedded OpenAPI specification at
runtime, derives kebab-case command names from operation IDs and maps path,
query and body parameters to string flags. Operations annotated with
`x-wade-cli-ignore` are excluded, and `wade api` lists the generated commands.

Requests target the first configured address among `--address`, the
development address when `WADE_DEV` is enabled, `WADE_ADDR`, the managed
daemon address and the standard local address. The controller never starts
the server. Successful response bodies stream unchanged to stdout, `204 No
Content` produces no output, and non-2xx responses surface the problem
payload through the process error path with a non-zero exit status.

## Config

The `config` controller ensures the settings file exists through the shared `settings` Model, then opens it in the selected editor. Editor selection, process IO and command-line presentation remain CLI concerns.

## Help

The `help` controller writes command usage and the available WADE commands. It has no Model or runtime dependencies.

## Lifecycle

The `lifecycle` controller handles the `start`, `status` and `stop` commands. It starts the managed background daemon, reports lifecycle state and the daemon's embedded build version, and requests graceful shutdown through `daemon`. The `start --foreground` mode runs the HTTP server directly without daemon management.

When running the HTTP server, the controller loads runtime configuration from `settings`, loads embedded web assets and constructs the application through `internal/app`. It owns listener setup, operating-system signal handling and graceful HTTP shutdown, while `internal/app` constructs the Models and HTTP controllers.
