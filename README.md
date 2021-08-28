# Imposter CLI

Features:

* run standalone mocks in place of real systems
* turn a OpenAPI/Swagger file into a mock API for testing or QA
* quickly set up a temporary API for your mobile/web client teams whilst the real API is being built
* decouple your integration tests from the cloud/various back-end systems and take control of your dependencies

Provide mock responses using static files or customise behaviour based on characteristics of the request. Power users can control mock responses with JavaScript or Java/Groovy script engines. Advanced users can write their own plugins in a JVM language of their choice.

## Getting started & documentation

### Installation

See the [Installation](./docs/install.md) instructions for your system.

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
  help        Help about any command
  mock        Start live mocks of APIs
```

Create and start mocks:

```
Starts a live mock of your APIs, using their Imposter configuration.

Usage:
  imposter mock [CONFIG_DIR]
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

Imposter is a reliable, scriptable and extensible mock server for general REST APIs, [OpenAPI](https://github.com/OAI/OpenAPI-Specification) (and Swagger) specifications, Salesforce and HBase APIs.

Scripting support for both JavaScript or Groovy/Java.

> Learn more about [Imposter](https://github.com/outofcoffee/imposter).

---

## Contributing

Suggestions and improvements to the CLI or documentation are welcome. Please raise pull requests targeting the `main` branch.
