ARG egover=1.6.1

FROM ghcr.io/edgelesssys/ego/build-base:v${egover}

RUN apt-get update \
&&  apt-get install -y --no-install-recommends \
wget gnupg ca-certificates build-essential libssl-dev libcurl4 git curl

# Configure DCAP
RUN mkdir -p /etc/apt/keyrings \
&&  wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | tee /etc/apt/keyrings/intel-sgx-keyring.asc > /dev/null \
&&  echo "deb [signed-by=/etc/apt/keyrings/intel-sgx-keyring.asc arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main" | tee /etc/apt/sources.list.d/intel-sgx.list \
&&  apt-get update \
&&  apt-get install -y libsgx-dcap-default-qpl

ARG egover
# Download and install EGo
# Use --force-depends to ignore SGX dependencies, which aren't required for building
RUN egodeb=ego_${egover}_amd64_ubuntu-$(grep -oP 'VERSION_ID="\K[^"]+' /etc/os-release).deb \
  && wget https://github.com/edgelesssys/ego/releases/download/v${egover}/${egodeb} \
  && dpkg -i --force-depends ${egodeb}

ENV CGO_CFLAGS=-I/opt/ego/include CGO_LDFLAGS=-L/opt/ego/lib

COPY sgx_default_qcnl.conf /etc/sgx_default_qcnl.conf

WORKDIR /app

ARG verifier_version=v2.2.0
RUN git clone -b ${verifier_version} --depth 1 https://github.com/zkportal/oracle-verification-backend .

RUN ego-go build

EXPOSE 8080

ENTRYPOINT ["/app/oracle-verification-backend"]
