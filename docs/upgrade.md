# Upgrade

Imposter can be [installed](./install.md) on Linux, macOS and Windows. This document explains how to upgrade Imposter to the latest version.

### Homebrew

If you installed Imposter using Homebrew, upgrade as follows:

    brew upgrade imposter

### Shell script

If you used the shell script approach (macOS and Linux only), you can re-run the script to upgrade

```shell
curl -L https://raw.githubusercontent.com/imposter-project/imposter-cli/main/install/install_imposter.sh | bash -
```

> **Warning**
> It is good practice to examine [the script](../install/install_imposter.sh) first.

See [Releases](https://github.com/imposter-project/imposter-cli/releases) for the latest version.

## Manual upgrade

### macOS

Only Intel x86_64 and ARM64 are supported on macOS.

```shell
# see https://github.com/imposter-project/imposter-cli/releases
export IMPOSTER_CLI_VERSION=0.1.0

curl -L -o imposter.tar.gz "https://github.com/imposter-project/imposter-cli/releases/download/v${IMPOSTER_CLI_VERSION}/imposter_${IMPOSTER_CLI_VERSION}_macOS_x86_64.tar.gz"
tar xvf imposter.tar.gz
mv ./imposter /usr/local/bin/imposter
```

### Linux

Intel x86_64, ARM32 and ARM64 is supported on Linux.

```shell
# see https://github.com/imposter-project/imposter-cli/releases
export IMPOSTER_CLI_VERSION=0.1.0

# choose one
#export IMPOSTER_ARCH=arm64
#export IMPOSTER_ARCH=arm
export IMPOSTER_ARCH=x86_64

curl -L -o imposter.tar.gz "https://github.com/imposter-project/imposter-cli/releases/download/v${IMPOSTER_CLI_VERSION}/imposter_${IMPOSTER_CLI_VERSION}_Linux_{IMPOSTER_ARCH}.tar.gz"
tar xvf imposter.tar.gz
mv ./imposter /usr/local/bin/imposter
```

### Windows

Only Intel x86_64 is supported on Windows.

> These instructions assume `curl` and `unzip` are available. You can also download the ZIP archive from the [Releases](https://github.com/imposter-project/imposter-cli/releases) page.

```
# see https://github.com/imposter-project/imposter-cli/releases
SET IMPOSTER_CLI_VERSION=0.1.0

curl.exe --output imposter.zip --url "https://github.com/imposter-project/imposter-cli/releases/download/v%IMPOSTER_CLI_VERSION%/imposter_%IMPOSTER_CLI_VERSION%_Windows_x86_64.zip"
unzip.exe imposter.zip

# use command (or add to PATH)
imposter.exe [command/args]
```
