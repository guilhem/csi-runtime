[![Go Reference](https://pkg.go.dev/badge/github.com/guilhem/csi-runtime.svg)](https://pkg.go.dev/github.com/guilhem/csi-runtime)

# csi-runtime

Simplify the creation of a CSI driver

## [driver](https://pkg.go.dev/github.com/guilhem/csi-runtime/driver)

Main entrypoint to create your driver.

## [identity](https://pkg.go.dev/github.com/guilhem/csi-runtime/identity)

Identity service allow the Orchestrator to query a plugin for capabilities, health, and other metadata

### Probe

In the `Probe` interface, you can code how Orchestrator will check the healthiness of your plugin.

## [controller](https://pkg.go.dev/github.com/guilhem/csi-runtime/controller)

Controller service aim to manage global calls, like creating a disk in the CSP API.

Controller is optional.

Current implementation is more than alpha and doesn't represent what this runtime should be.

## [node](https://pkg.go.dev/github.com/guilhem/csi-runtime/node)

Node manager is the code running on each node that is reponsible for making volumes available.

_csi-runtime_ abstract all the staging / publishing part and let you focus on your work.

### Validate

In the `Validate` interface, you can code the volume capability and reject if necessary.

### Block

In the `Block` interface, you can code how you are attaching and detaching the block device on the node.

### Mount

In the `Mount` interface, you can send the options for mounting a filesystem.
It can be the previous device or something else (like fuse).

### Populate

In the `Populate` interface, you can create any file you want in a particular path.
