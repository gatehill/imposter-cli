# Using the unpacked mock engine

> **Important:** This method is primarily intended for use by tools, not end users. If you are unsure, and you really want to use the JVM directly, you probably want the [JVM engine](./jvm_engine.md). 

Imposter supports different mock engine types: [Docker](./docker_engine.md) and [JVM](./jvm_engine.md). This document describes how to use the **unpacked** engine.

## Prerequisites

- Install a Java 8+ JVM.
- Obtain an unpacked distribution of Imposter.

The unpacked distibution must be structured as follows:

```
/path/to/distro
\- lib
   \- imposter-api.jar
   \- imposter-engine.jar
   \- ...
```

## Configuration

### User default

The easiest way to set the engine type is to edit your user default [configuration](./config.md) in:

    $HOME/.imposter/config.yaml

Set the `engine` key to `unpacked` and set the distribution directory:

```yaml
engine: unpacked
jvm:
  distroDir: /path/to/distro
```

### Environment variable

If you don't want to set your user defaults you can set the following environment variables:

    IMPOSTER_ENGINE=unpacked
    IMPOSTER_JVM_DISTRO_DIR=/path/to/distro

### Command line argument

You can also provide the `--engine-type` (or `-t`) command line argument to the `imposter up` command.

> **Important** You must also set the distribution directory using either the configuration or environment variables described above.

Example:

    imposter up --engine-type unpacked

Or:

    imposter up -t unpacked
