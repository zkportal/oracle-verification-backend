# Copyright (C) 2023 zkPortal. All rights reserved.
# Copyright (C) 2021 Edgeless Systems GmbH. All rights reserved.
# Copyright (C) 2011-2021 Intel Corporation. All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#
#   * Redistributions of source code must retain the above copyright
#     notice, this list of conditions and the following disclaimer.
#   * Redistributions in binary form must reproduce the above copyright
#     notice, this list of conditions and the following disclaimer in
#     the documentation and/or other materials provided with the
#     distribution.
#   * Neither the name of Intel Corporation nor the names of its
#     contributors may be used to endorse or promote products derived
#     from this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
# "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
# LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
# A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
# OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
# SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
# LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
# DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
# THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

FROM node:18.18.2-bookworm AS builder

# DCAP version (github repo branch, tag or commit hash)
ARG DCAP_VERSION=DCAP_1.19

# update and install packages
RUN DEBIAN_FRONTEND=noninteractive \
    apt-get update -yq && \
    apt-get upgrade -yq && \
    apt-get install -yq --no-install-recommends \
    apt-utils \
    autoconf \
    automake \
    build-essential \
    ca-certificates \
    curl \
    git \
    zip \
    libcurl4-openssl-dev \
    pkgconf \
    libboost-dev \
    libboost-system-dev \
    libboost-thread-dev \
    protobuf-c-compiler \
    libprotobuf-c-dev \
    protobuf-compiler \
    ocaml ocamlbuild libtool wget python-is-python3 libssl-dev cmake perl

RUN apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/intel

# clone SGX SDK repo
RUN git clone https://github.com/intel/linux-sgx.git -b sgx_2.22 --depth 1

WORKDIR /opt/intel/linux-sgx

# build SGX SDK and install it to /opt/intel/sgxsdk
RUN make preparation && make sdk_install_pkg && ./linux/installer/bin/sgx_linux_x64_sdk_2.22.100.3.bin --prefix=/opt/intel/

WORKDIR /opt/intel

# clone DCAP repo
RUN git clone https://github.com/intel/SGXDataCenterAttestationPrimitives.git -b ${DCAP_VERSION} --depth 1

# set PWD to PCKCertSelection dir
WORKDIR /opt/intel/SGXDataCenterAttestationPrimitives

# build libPCKCertSelection library and copy to lib folder
RUN make PCKCertSelection && \
    mkdir -p ./QuoteGeneration/pccs/lib && \
    cp ./tools/PCKCertSelection/out/libPCKCertSelection.so ./QuoteGeneration/pccs/lib/ && \
    make clean

# set PWD to PCCS dir
WORKDIR /opt/intel/SGXDataCenterAttestationPrimitives/QuoteGeneration/pccs/

# build pccs
RUN npm config set engine-strict true && \
    npm install

# build final image
FROM node:18.18.2-bookworm

RUN apt-get update && \
    apt-get install -yq --no-install-recommends openssl ca-certificates

ARG USER=pccs
ARG UID=65333

# create user and a group
RUN useradd -M -U ${USER} --uid=${UID} -s /bin/false

COPY --from=builder --chown=${USER}:${USER} /opt/intel/SGXDataCenterAttestationPrimitives/QuoteGeneration/pccs/ /opt/intel/pccs/

WORKDIR /opt/intel/pccs/
USER ${USER}

# copy script to setup keys
COPY setup.sh /opt/intel/pccs/setup.sh

# entrypoint to start pccs
ENTRYPOINT ["sh", "/opt/intel/pccs/setup.sh"]
