---
sidebar_position: 1
---

# Getting started

Firstly, [install](./installation.md) the bx2cloud CLI.

Make sure you have a running [bx2cloud API](../api/installation.md) instance.

You should then be able to use the `bx2cloud` command. Here are some examples:

```sh
$ bx2cloud network get 3
id  internetAccess
3   true

$ bx2cloud subnetwork delete 5
Successfully deleted 5

$ bx2cloud container list
id  image         status        ip
2   nginx:latest  running (3s)  10.0.42.2/24
```
