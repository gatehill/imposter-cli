# Using the JVM mock engine

Imposter supports different mock engine types: [Docker](./docker_engine.md) and JVM. This document describes how to use the **JVM** engine.

## Prerequisites

Install a Java 8+ JVM.

For example, if you are using Homebrew install with:

    brew install openjdk@11

Or choose a distribution of your choice, such as [Eclipse Adoptium](https://adoptium.net/).

## Configuration

### User default

The easiest way to set the engine type is to edit your user default [configuration](./config.md) in:

    $HOME/.imposter/config.yaml

Set the `engine` key to `jvm`:

```yaml
engine: jvm
```

### Command line argument

If you don't want to set your user defaults you can provide the `--engine` (or `-e`) argument to the `imposter up` command:

Example:

    imposter up --engine jvm

Or:

    imposter up -e jvm