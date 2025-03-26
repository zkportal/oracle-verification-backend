#!/bin/sh

# configurable options using env
TEMP_WD=$TEMP_WD
ORACLE_REVISION=$ORACLE_REVISION

START_DIR=$(pwd)

# set default temp directory
if [ "$TEMP_WD" = "" ]; then
  TEMP_WD=$(mktemp -d -p $START_DIR)
fi

# set default source code revision to master
if [ "$ORACLE_REVISION" = "" ]; then
  ORACLE_REVISION="main"
fi

finish() {
  result=$?

  if [ "$result" -ne "0" ]; then
    echo "Failed to obtain Oracle's unique ID"
  fi

  # clean up temp dir
  if [ "$TEMP_WD" != "" ]; then
    rm -rf "$TEMP_WD"
  fi

  exit $result
}

usage() {
  echo "Aleo Oracle verification script for getting Oracle backend SGX and Nitro enclave measurements using reproducible build."
  echo ""
  echo "This script essentially is the process described in https://github.com/zkportal/oracle-notarization-backend?tab=readme-ov-file#reproducible-build but without the installation steps"
  echo ""
  echo "This script accepts some configuration options using environment variables:"
  echo "\t - TEMP_WD - path to a temporary directory where the script will be downloading, will be deleted automatically. Optional, uses current working directory by default."
  echo "\t - ORACLE_REVISION - Oracle backend source code revision to check out. Optional, uses master branch by default."
  echo ""
  echo "Example: ORACLE_REVISION=v1.8.0 ./get-enclave-id.sh"
  echo ""
  echo "Script dependencies:"
  echo "\t- Docker with Buildkit"
  echo "\t - OpenSSL"
  echo "\t- jq"
}

help_wanted() {
  [ "$#" -ge "1" ] && { [ "$1" = "-h" ] || [ "$1" = "--help" ] || [ "$1" = "-?" ]; };
}

check_dependencies() {
  openssl_version=$(openssl version)
  openssl_found=$?

  docker_version=$(docker --version)
  docker_found=$?

  jq_help=$(jq --help)
  jq_found=$?

  should_exit=0

  if [ "$openssl_found" -ne "0" ]; then
    echo "OpenSSL not found, exiting"
    should_exit=1
  fi

  if [ "$docker_found" -ne "0" ]; then
    echo "Docker not found, exiting"
    should_exit=1
  fi

  if [ "$jq_found" -ne "0" ]; then
    echo "JQ not found, exiting"
    should_exit=1
  fi

  # build nitro-cli image
  docker_build_output=$(docker build -qq -t nitro-cli -f Dockerfile.nitro .)
  nitro_cli_build_success=$?

  if [ "$nitro_cli_build_success" -ne "0" ]; then
    echo "failed to build nitro-cli image, exiting"
    echo "$docker_build_output"
    should_exit=1
  fi

  if [ "$should_exit" = 1 ]; then
    exit 1
  fi
}

trap finish EXIT

if help_wanted "$@"; then
  usage
  exit 0
fi

echo "Using temp directory $TEMP_WD"

check_dependencies

(
  cd $TEMP_WD

  echo "Building Oracle backend source code ($ORACLE_REVISION) image..."
  # Build a source code image first. This step may be removed in the future if we publish images
  docker_build_output=$(DOCKER_BUILDKIT=1 docker build -qq --network host --platform linux/amd64 --build-arg VERSION=$ORACLE_REVISION -f ../Dockerfile.oracle.source -t oracle-notarization-backend-src:$ORACLE_REVISION .)
  docker_build=$?

  if [ "$docker_build" -ne "0" ]; then
    echo "Failed to build Oracle backend source code image"
    echo "$docker_build_output"
    exit 1
  fi

  echo "Generating a temp signing key for the SGX enclave..."
  openssl_keygen=$(openssl genrsa -out private.pem -3 3072)
  openssl_keygen_result=$?

  if [ "$openssl_keygen_result" != 0 ]; then
    echo "Failed to generate a temporary signing key for the SGX enclave."
    echo "$openssl_keygen"
    exit 1
  fi

  echo "Building Oracle backend SGX enclave..."

  # Build SGX enclave
  sgx_build=$(DOCKER_BUILDKIT=1 docker build -qq --network host --build-arg "egover=1.6.1" --build-arg "oracle_version=$ORACLE_REVISION" -f ../Dockerfile.oracle.sgx --secret id=signingkey,src=private.pem -o . .)
  sgx_build_result=$?

  if [ "$sgx_build_result" != 0 ]; then
    echo "Failed to build Oracle backend enclave. There may be a problem with EGo. If not, try a different revision."
    echo "$sgx_build"
    exit 1
  fi

  # uniqueid.txt is exported from Dockerfile.oracle.sgx
  unique_id=$(cat uniqueid.txt)
  echo "Oracle SGX unique ID:"
  echo "$unique_id"

  echo "Building Oracle backend Nitro image..."

  # Build Nitro enclave (Dockerfile.nitro from the notarization backend sources, not the one in this repo)
  docker_build_output=$(docker build --network=host -qq -f Dockerfile.nitro -t oracle-notarization-backend .)
  docker_build=$?

  if [ "$docker_build" -ne "0" ]; then
    echo "Failed to build Oracle backend docker image"
    echo "$docker_build_output"
    exit 1
  fi

  echo "Building Oracle backend Nitro enclave..."

  nitro_build=$(docker run --name nitro-cli-build --rm -v /var/run/docker.sock:/var/run/docker.sock nitro-cli oracle-notarization-backend)
  enclave_build_success=$?

  if [ "$enclave_build_success" -ne "0" ]; then
    echo "Failed to build Nitro Oracle backend enclave. Output:"
    echo "$nitro_build"
    exit 1
  fi

  echo "Oracle Nitro PCR:"
  echo "$nitro_build" | jq -r '.Measurements.PCR0'
  echo "$nitro_build" | jq -r '.Measurements.PCR1'
  echo "$nitro_build" | jq -r '.Measurements.PCR2'

  exit 0
)
