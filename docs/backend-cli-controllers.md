# CLI Controllers

Command-line controllers live in `cmd/wade/internal/controllers` and own argument handling, process output and lifecycle presentation. The CLI router dispatches commands through a common `HandleArgs` interface and injects the shared `settings` Model where required.

WADE's CLI controller set includes:

- [`config`](#config)
- [`help`](#help)
- [`lifecycle`](#lifecycle)

## Config

The `config` controller ensures the settings file exists through the shared `settings` Model, then opens it in the selected editor. Editor selection, process IO and command-line presentation remain CLI concerns.

## Help

The `help` controller writes command usage and the available WADE commands. It has no Model or runtime dependencies.

## Lifecycle

The `lifecycle` controller handles the `start`, `status` and `stop` commands. It starts the managed background daemon, reports lifecycle state and the daemon's embedded build version, and requests graceful shutdown through `daemon`. The `start --foreground` mode runs the HTTP server directly without daemon management.

When running the HTTP server, the controller loads runtime configuration from `settings`, loads embedded web assets and constructs the application through `internal/app`. It owns listener setup, operating-system signal handling and graceful HTTP shutdown, while `internal/app` constructs the Models and HTTP controllers.
