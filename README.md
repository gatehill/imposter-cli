# Imposter: A scriptable, multipurpose mock server

Reliable, scriptable and extensible mock server for general REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs. 

Features:

* run standalone mocks in place of real systems
* turn a OpenAPI/Swagger file into a mock API for testing or QA
* quickly set up a temporary API for your mobile/web client teams whilst the real API is being built
* decouple your integration tests from the cloud/various back-end systems and take control of your dependencies

Provide mock responses using static files or customise behaviour based on characteristics of the request.
Capture data and use response templates to provide conditional responses.
Power users can control mock responses with JavaScript or Java/Groovy script engines.

> This project is the CLI tool for the [Imposter mock engine](https://github.com/outofcoffee/imposter).

## Getting started & documentation

### Installation

See the [Installation](./docs/install.md) instructions for your system.

Or, use this one liner (macOS and Linux only):

```shell
curl -L https://raw.githubusercontent.com/gatehill/imposter-cli/main/install/install_imposter.sh | bash -
```

> Note: You must have [Docker](https://docs.docker.com/get-docker/) installed.

### Usage

Top level command:

```
Usage:
  imposter [command]

Available Commands:
  up          Start live mocks of APIs
  help        Help about any command
```

Create and start mocks:

```
Starts a live mock of your APIs, using their Imposter configuration.

Usage:
  imposter up [CONFIG_DIR] [flags]

Flags:
  -p, --port int         Port on which to listen (default 8080)
  -v, --version string   Imposter engine version (default "latest")
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

Imposter is a reliable, scriptable and extensible mock server for general REST APIs, OpenAPI (and Swagger) specifications, Salesforce and HBase APIs.

Scripting support for both JavaScript or Groovy/Java.

> Learn more about [Imposter](https://github.com/outofcoffee/imposter).

---

## Contributing

Suggestions and improvements to the CLI or documentation are welcome. Please raise pull requests targeting the `main` branch.
