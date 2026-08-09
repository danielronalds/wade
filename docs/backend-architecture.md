# Backend Architecture

## Overview

WADE's backend uses a direct MVC-style structure: `controllers -> models -> infrastructure`. HTTP and WebSocket controllers live in `internal/controllers`, aggregate Models live in `internal/models/<aggregate>`, and external IO implementations live in `internal/infrastructure/<capability>`. Application wiring is split between `cmd/wade/main.go`, the top-level composition root, and `internal/app`, which constructs the HTTP application.

Dependencies flow in one direction. Controllers handle transport concerns and coordinate Models, Models own domain behaviour, validation, state and concurrency, and infrastructure performs mechanical filesystem, environment, Git, provider and PTY operations. Infrastructure never depends on Models or controllers, Models never depend on controllers, and cross-aggregate workflows are composed at the controller layer rather than through Model-to-Model dependencies.

### Dependency Flow

```mermaid
flowchart TB
    subgraph Command["cmd/wade"]
        direction TB
        Main["main.go<br/>Top-level composition"]
        Router["CLI router"]
        Commands["CLI controllers<br/>config, help and server lifecycle"]
        Main --> Router --> Commands
    end

    subgraph Core["internal/"]
        direction TB
        App["app<br/>HTTP application composition"]
        Server["server<br/>HTTP routing"]
        Controllers["controllers<br/>Transport and orchestration"]
        Models["models<br/>Domain behaviour and state"]
        Infrastructure["infrastructure<br/>External IO"]
        Daemon["daemon<br/>Background server lifecycle"]

        App --> Server --> Controllers --> Models --> Infrastructure
    end

    Commands --> App
    Commands --> Daemon

    style Core stroke-dasharray: 5 5
```

## Architecture Details

- [HTTP Controllers](backend-controllers.md) describes API transport boundaries, Model orchestration and error mapping.
- [CLI Controllers](backend-cli-controllers.md) describes command routing and server lifecycle operations.
- [Aggregate Models](backend-models.md) describes domain ownership, state, concurrency and cross-aggregate boundaries.
- [Infrastructure Modules](backend-infrastructure.md) describes external IO capabilities and their boundary with Models.
