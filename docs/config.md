# Configuration

You can configure the Imposter CLI using command line arguments/flags or configuration files.

## Command line

Each command has its own list of arguments and flags, accessible using the `-h` flag, such as:

```
$ imposter up -h
Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.

Usage:
  imposter up [CONFIG_DIR] [flags]

Flags:
      --auto-restart     Automatically restart when config dir contents change (default true)
  -e, --engine string    Imposter engine type (docker|jvm - default docker)
  -h, --help             help for up
  -p, --port int         Port on which to listen (default 8080)
      --pull             Force engine pull
  -v, --version string   Imposter engine version (default "latest")
```

## Mock configuration files

Mocks are configured using files with the following suffixes:

* `-config.yaml`
* `-config.yml`
* `-config.json`

> For example: `orders-mock-config.yaml`

These files control behaviour such as responses, validation, scripting and more.

Learn about [Imposter mock configuration](https://docs.imposter.sh/configuration/) files.

## CLI Configuration file

You can also use a configuration file to set CLI defaults. By default, Imposter looks for a CLI configuration file located at `$HOME/.imposter/config.yaml`

> You can override the path to the CLI configuration file by passing the `--config CONFIG_PATH` flag.

The currently supported elements are as follows:

```yaml
# the engine type - valid values are "docker" or "jvm"
engine: "docker"

# the engine version - valid values are "latest", or a binary release such as "1.21.0"
# see: https://github.com/outofcoffee/imposter/releases
version: "latest"
```

### Engine types

Imposter supports different mock engine types: Docker (default) and JVM. For more information about configuring the engine type see:

- [Docker engine](./docker_engine.md) (default)
- [JVM engine](./jvm_engine.md)
