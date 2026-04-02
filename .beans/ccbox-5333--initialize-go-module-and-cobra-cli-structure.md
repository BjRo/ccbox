---
# ccbox-5333
title: Initialize Go module and Cobra CLI structure
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:33:55Z
updated_at: 2026-04-02T10:33:55Z
parent: ccbox-jxut
---

## Description
Initialize the Go module (`github.com/bjro/ccbox`), set up Cobra CLI with a root command and `init` subcommand. Establish the project directory structure:

```
cmd/
  root.go        # Root Cobra command
  init.go        # `ccbox init` subcommand
internal/
  detect/        # Stack detection
  template/      # Template engine
  firewall/      # Domain allowlist logic
  config/        # .ccbox.yml handling
main.go
```

Use Go 1.24+ with modules. Add a basic `--version` flag.