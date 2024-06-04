FROM ubuntu:focal

WORKDIR /app

RUN apt update \
&&  apt install -y --no-install-recommends \
    wget gnupg ca-certificates build-essential libssl-dev golang-1.21 libcurl4 git curl \
&&  ln -s /usr/lib/go-1.21/bin/go /usr/local/bin/go

# install EGo
RUN mkdir -p /etc/apt/keyrings \
&&  wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | tee /etc/apt/keyrings/intel-sgx-keyring.asc > /dev/null \
&&  echo "deb [signed-by=/etc/apt/keyrings/intel-sgx-keyring.asc arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main" | tee /etc/apt/sources.list.d/intel-sgx.list \
&&  apt update \
&&  wget -q https://github.com/edgelesssys/ego/releases/download/v1.5.2/ego_1.5.2_amd64_ubuntu-20.04.deb \
&&  apt install -y ./ego_1.5.2_amd64_ubuntu-20.04.deb libsgx-dcap-default-qpl

ENV CGO_CFLAGS=-I/opt/ego/include CGO_LDFLAGS=-L/opt/ego/lib

COPY sgx_default_qcnl.conf /etc/sgx_default_qcnl.conf

ADD . .

RUN go mod download

EXPOSE 8080

ENTRYPOINT ["go", "run", "main.go"]
