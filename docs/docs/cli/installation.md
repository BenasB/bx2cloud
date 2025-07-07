---
sidebar_position: 2
---

# Installation

Installing bx2cloud CLI can be done in multiple ways:

1. Binary download
2. Container image
3. Building from source

### 1. Binary download

It's possible to download a pre-built binary of the CLI from [GitHub releases](https://github.com/BenasB/bx2cloud/releases). CLI builds start with `bx2cloud_`. Name it `bx2cloud` and place it somewhere on your `PATH`.

### 2. Container image

A container image is available on [Docker Hub](https://hub.docker.com/r/benasbudrys/bx2cloud-cli).

Running `bx2cloud container list` can be achieved with:

```sh
docker run benasbudrys/bx2cloud-cli -t <api-ip>:<api-port> container list
```

### 3. Building from source

You can install the CLI using `go`:

```sh
go install github.com/BenasB/bx2cloud/cmd/bx2cloud@latest
```

Please check the required go version in [go.mod](https://github.com/BenasB/bx2cloud/blob/main/go.mod).
