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

Learn about [Imposter mock configuration](https://outofcoffee.github.io/imposter/configuration.html) files.

## CLI Configuration file

You can also use a configuration file to set CLI defaults. The configuration file should be located at `$HOME/.imposter/config.yaml`

The currently supported elements are as follows:

````yaml
# the engine type - valid values are "docker" or "jvm"
engine: "docker"

# the engine version - valid values are "latest", or a binary release such as "1.21.0"
# see: https://github.com/outofcoffee/imposter/releases
version: "latest"
````
