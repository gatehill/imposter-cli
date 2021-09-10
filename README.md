# Imposter: A scriptable, multipurpose mock server

Reliable, scriptable and extensible mock server for REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs. 

Features:

* run standalone mocks in place of real systems
* turn an OpenAPI/Swagger file into a mock API for testing or QA
* quickly set up a temporary API for your mobile/web client teams whilst the real API is being built
* decouple your integration tests from the cloud/various back-end systems and take control of your dependencies
* validate your API requests against an OpenAPI specification

Provide mock responses using static files or customise behaviour based on characteristics of the request.
Capture data and use response templates to provide conditional responses.

Power users can control mock responses with JavaScript or Java/Groovy script engines.
Advanced users can write their own plugins in a JVM language of their choice.

> This project is the CLI tool for the [Imposter mock engine](https://github.com/outofcoffee/imposter).

<img src="./docs/img/imposter-scaffold.gif" alt="Screenshot of scaffold command" width="67%">

## Getting started & documentation

You must have [Docker](https://docs.docker.com/get-docker/) installed.

### Installation

See the [Installation](./docs/install.md) instructions for your system.

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
  help        Help about any command
  version     Print CLI version
```

Create and start mocks:

```
Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.

Usage:
  imposter up [CONFIG_DIR] [flags]

Flags:
  -p, --port int         Port on which to listen (default 8080)
      --pull             Force engine image pull
      --auto-restart     Automatically restart when config dir contents change (default true)
  -v, --version string   Imposter engine version (default "latest")
```

Scaffold Imposter configuration from OpenAPI specification files:
```
Creates Imposter configuration from one or more OpenAPI/Swagger specification files.

If CONFIG_DIR is not specified, the current working directory is used.

Usage:
  imposter scaffold [CONFIG_DIR] [flags]

Flags:
  -f  --force-overwrite        Force overwrite of destination file(s) if already exist
      --generate-resources     Generate Imposter resources from OpenAPI paths (default true)
  -s  --script-engine string   Generate placeholder Imposter script (none|groovy|js) (default "none")
```

Help:

```
Provides help for any command in the application.
Simply type imposter help [path to command] for full details.

Usage:
  imposter help [command] [flags]
```

### Logging

The default log level is `debug`. You can override this by setting the `LOG_LEVEL` environment variable:

    export LOG_LEVEL=info

---

## About Imposter

[Imposter](https://github.com/outofcoffee/imposter) is a reliable, scriptable and extensible mock server for REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs.

Scripting support for both JavaScript or Groovy/Java.

### User documentation

**[Read the user documentation here](https://outofcoffee.github.io/imposter/)**

---

## Contributing

Suggestions and improvements to the CLI or documentation are welcome. Please raise pull requests targeting the `main` branch.
