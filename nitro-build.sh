#!/bin/bash

/usr/bin/nitro-cli build-enclave --output-file /app/enclave.eif --docker-uri $1 2>1
