---
sidebar_position: 1
---

# Installation

Installing bx2cloud API can be done in multiple ways:

1. Binary download
2. Container image
3. Building from source

#### Requirements

- The API supports running only on Linux, since most of the functionality depends on it (such as linux namespaces or networking). Linux specific requirements include:
  - iptables
- It also requires root privileges (to create linux namespaces, set up network routes, enable certain sysctl options).

### 1. Binary download

It's possible to download a pre-built binary of the API from GitHub releases. WIP 🚧

### 2. Container image

A container image is available on Docker Hub. WIP 🚧

### 3. Building from source

You can install the API using `go`:

```sh
go install github.com/BenasB/bx2cloud/cmd/bx2cloud-api@latest
```

Please check the required go version in [go.mod](https://github.com/BenasB/bx2cloud/blob/main/go.mod).

Or clone the bx2cloud GitHub [repository](https://github.com/BenasB/bx2cloud) and build it manually:

```sh
go build ./cmd/bx2cloud-api
```

:::info

`CGO_ENABLED` must be set to `1` when building the API because of the dependency on [libcontainer/nsenter](https://pkg.go.dev/github.com/opencontainers/runc@v1.3.0/libcontainer/nsenter). You can check the current value with `go env CGO_ENABLED`

:::
