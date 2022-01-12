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
      --auto-restart         Automatically restart when config dir contents change (default true)
      --deduplicate string   Override deduplication ID for replacement of containers
      --enable-plugins       Whether to enable plugins (default true)
  -t, --engine-type string   Imposter engine type (valid: docker,jvm - default "docker")
  -e, --env stringArray      Explicit environment variables to set
  -h, --help                 help for up
  -p, --port int             Port on which to listen (default 8080)
      --pull                 Force engine pull
  -s, --scaffold             Scaffold Imposter configuration for all OpenAPI files
  -v, --version string       Imposter engine version (default "latest")
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

# the engine version - valid values are "latest", or a binary release such as "2.0.1"
# see: https://github.com/outofcoffee/imposter/releases
version: "latest"

# Docker engine specific configuration
docker:
  # bind mount flags
  # see: https://docs.docker.com/storage/bind-mounts
  bindFlags: ":z"

  # the container user (username or uid)
  containerUser: "imposter"

# JVM engine specific configuration
jvm:
  # override the path to the Imposter JAR file to use (default: automatically generated)
  jarFile: "/path/to/imposter.jar"
  
  # directory holding the JAR file cache (default: "$HOME/.imposter/cache")
  binCache: "/path/to/dir"

  # directory containing an unpacked Imposter distribution
  # note: this is generally only used by other tools
  distroDir: "/path/to/unpacked/distro"

# Plugin configuration
plugin:
  # base directory holding plugin files (default: "$HOME/.imposter/plugins")
  baseDir: "/path/to/dir"
```

## Environment variables

Some configuration elements can be specified as environment variables:

* IMPOSTER_CLI_LOG_LEVEL
* IMPOSTER_ENGINE
* IMPOSTER_VERSION
* IMPOSTER_DOCKER_BINDFLAGS
* IMPOSTER_DOCKER_CONTAINERUSER
* IMPOSTER_JVM_JARFILE
* IMPOSTER_JVM_BINCACHE
* IMPOSTER_JVM_DISTRODIR
* IMPOSTER_PLUGIN_BASEDIR

### Engine types

Imposter supports different mock engine types: Docker (default) and JVM. For more information about configuring the engine type see:

- [Docker engine](./docker_engine.md) (default)
- [JVM engine](./jvm_engine.md)
