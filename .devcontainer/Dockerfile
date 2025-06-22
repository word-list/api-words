FROM mcr.microsoft.com/dotnet/sdk:8.0

RUN apt-get update && apt-get install -y \
    clang \
    lld \
    libz-dev \    
    libc6-dev-arm64-cross \
    gcc-aarch64-linux-gnu \
    ssh-client \
    && rm -rf /var/lib/apt/lists/*

ENV PATH="${PATH}:/root/.dotnet/tools"
RUN dotnet tool install --global dotnet-ef --version 9.*

RUN curl https://binaries.cockroachdb.com/cockroach-v23.1.0.linux-amd64.tgz | tar -xz && \
    mv cockroach-v23.1.0.linux-amd64/cockroach /usr/local/bin/

WORKDIR /workspace