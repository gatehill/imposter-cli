# Imposter: A scriptable, multipurpose mock server

Reliable, scriptable and extensible mock server for REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs. 

Start a live mock of an OpenAPI specification with just:

```shell
$ imposter up -s

found 1 OpenAPI spec(s)
starting server on port 8080...
...
mock server up and running at http://localhost:8080
```

Features:

- run standalone mocks in place of real systems
- turn an OpenAPI/Swagger file into a mock API for testing or QA (even before the real API is built)
- decouple your integration tests from the cloud/various back-end systems and take control of your dependencies
- validate your API requests against an OpenAPI specification
- capture data and validate later or use response templates to provide conditional responses

Send dynamic responses:

- Provide mock responses using static files or customise behaviour based on characteristics of the request.
- Power users can control mock responses with JavaScript or Java/Groovy script engines.
- Advanced users can write their own plugins in a JVM language of their choice.

> This project is the CLI tool for the [Imposter mock engine](https://github.com/outofcoffee/imposter).

You can also generate Imposter configuration from OpenAPI files:

<img src="./docs/img/imposter-scaffold.gif" alt="Screenshot of scaffold command" width="67%">

## Getting started & documentation

You must have [Docker](https://docs.docker.com/get-docker/) installed and running.

### Installation

See the [Installation](./docs/install.md) instructions for your system or follow the quick-start instructions below:

#### Homebrew

If you have Homebrew installed:

    brew tap gatehill/imposter
    brew install imposter

#### Shell script

Or, use this one liner (macOS and Linux only):

```shell
curl -L https://raw.githubusercontent.com/gatehill/imposter-cli/main/install/install_imposter.sh | bash -
```

### Usage

Top level command:

```
Usage:
  imposter [command]

Available Commands:
  up          Start live mocks of APIs
  scaffold    Create Imposter configuration from OpenAPI specs
  pull        Pull the engine into the cache
  doctor      Check prerequisites for running Imposter
  version     Print CLI version
  help        Help about any command
```

#### Create and start mocks

Example:

    imposter up

Usage:

```
Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.

Usage:
  imposter up [CONFIG_DIR] [flags]

Flags:
      --auto-restart     Automatically restart when config dir contents change (default true)
  -e, --engine string    Imposter engine type (valid: docker,jvm - default "docker")
  -p, --port int         Port on which to listen (default 8080)
      --pull             Force engine pull
  -s, --scaffold         Scaffold Imposter configuration for all OpenAPI files
  -v, --version string   Imposter engine version (default "latest")
```

#### Generate Imposter configuration from OpenAPI specification files

Example:

    imposter scaffold

Usage:

```
Creates Imposter configuration from one or more OpenAPI/Swagger specification files
in a directory.

If DIR is not specified, the current working directory is used.

Usage:
  imposter scaffold [DIR] [flags]

Flags:
  -f  --force-overwrite        Force overwrite of destination file(s) if already exist
      --generate-resources     Generate Imposter resources from OpenAPI paths (default true)
  -s  --script-engine string   Generate placeholder Imposter script (none|groovy|js) (default "none")
```

#### Pull engine

Example:

    imposter engine pull

Usage:

```
Pulls a specified version of the engine binary/image into the cache.

If version is not specified, it defaults to 'latest'.

Usage:
  imposter engine pull [flags]

Flags:
  -e, --engine string    Imposter engine type (valid: docker,jvm - default "docker")
  -h, --help             help for pull
      --pull             Force engine pull
  -v, --version string   Imposter engine version (default "latest")
```

#### Doctor

```
Checks prerequisites for running Imposter, including those needed
by the engines.

Usage:
  imposter doctor
```

#### Help

```
Provides help for any command in the application.
Simply type imposter help [path to command] for full details.

Usage:
  imposter help [command] [flags]
```

### Logging

The default log level is `debug`. You can override this by setting the `LOG_LEVEL` environment variable:

    export LOG_LEVEL=info

### Configuration

Learn more about [configuration](./docs/config.md).

---

## About Imposter

[Imposter](https://github.com/outofcoffee/imposter) is a reliable, scriptable and extensible mock server for REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs.

Scripting support for both JavaScript or Groovy/Java.

### User documentation

**[Read the user documentation here](https://outofcoffee.github.io/imposter/)**

---

## Contributing

Suggestions and improvements to the CLI or documentation are welcome. Please raise pull requests targeting the `main` branch.
