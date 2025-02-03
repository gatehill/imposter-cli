#!/usr/bin/env bash

# Copyright © 2021 Pete Cornish <outofcoffee@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Imposter CLI installation
# See: https://github.com/imposter-project/imposter-cli

set -e

BASE_URL="https://github.com/imposter-project/imposter-cli/releases"

function unsupported_arch() {
  echo "This OS/architecture is unsupported."
  exit 1
}

function is_macos() {
  case "$(uname -s)" in
  *darwin* ) true ;;
  *Darwin* ) true ;;
  * ) false;;
  esac
}

function is_linux() {
  case "$(uname -s)" in
  *Linux* ) true ;;
  *linux* ) true ;;
  * ) false;;
  esac
}

function find_arch() {
  if is_macos; then
    case "$(uname -m)" in
    *i386* ) IMPOSTER_ARCH="amd64" ;;
    *x86_64* ) IMPOSTER_ARCH="amd64" ;;
    *arm64* ) IMPOSTER_ARCH="arm64" ;;
    * ) unsupported_arch;;
    esac
  else
    case "$(uname -m)" in
    *i686* ) IMPOSTER_ARCH="amd64" ;;
    *x86_64* ) IMPOSTER_ARCH="amd64" ;;
    *armv6* ) IMPOSTER_ARCH="arm" ;;
    *armv7* ) IMPOSTER_ARCH="arm" ;;
    *arm64* ) IMPOSTER_ARCH="arm64" ;;
    *aarch64* ) IMPOSTER_ARCH="arm64" ;;
    * ) unsupported_arch;;
    esac
  fi
}

function find_os() {
    if is_macos; then
      IMPOSTER_OS="darwin"
    elif is_linux; then
      IMPOSTER_OS="linux"
    else
      unsupported_arch
    fi
}

function set_download_url() {
    local BINARY_NAME="imposter-cli_${IMPOSTER_OS}_${IMPOSTER_ARCH}.tar.gz"
    if [[ -z "${IMPOSTER_CLI_VERSION}" ]]; then
      echo "Using latest version..."
      DOWNLOAD_URL="${BASE_URL}/latest/download/${BINARY_NAME}"
    else
      echo "Using version: ${IMPOSTER_CLI_VERSION}"
      DOWNLOAD_URL="${BASE_URL}/download/v${IMPOSTER_CLI_VERSION}/${BINARY_NAME}"
    fi
}

find_os
find_arch
set_download_url

IMPOSTER_TEMP_DIR="$( mktemp -d /tmp/imposter-cli.XXXXXXX )"
cd "${IMPOSTER_TEMP_DIR}"

echo -e "\nDownloading from ${DOWNLOAD_URL}"
curl --fail -L -o imposter-cli.tar.gz "${DOWNLOAD_URL}"
tar xf imposter-cli.tar.gz

echo -e "\nInstalling to /usr/local/bin"
cp ./imposter /usr/local/bin/imposter

echo -e "\nDone"
