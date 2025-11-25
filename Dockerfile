# Stage 1: Build the Go project
FROM golang:1.23-alpine AS go-builder

# Use build arguments for the target architecture
ARG TARGETARCH
ARG GOARCH
ARG VERSION
ARG COMMIT

# See https://github.com/initia-labs/movevm/releases
ENV LIBMOVEVM_VERSION=v1.1.1
ENV ROCKS_DB_VERSION=v10.5.1

# Install necessary packages
RUN set -eux; apk add --no-cache ca-certificates build-base git cmake curl 

# Install dependencies for RocksDB
RUN apk add --update --no-cache snappy-dev zlib-dev bzip2-dev lz4-dev zstd-dev jemalloc-dev linux-headers
# Ensure toolchain variables are set explicitly for RocksDB build
ENV CC=gcc CXX=g++

WORKDIR /code
COPY . /code/

# Install rocksdb (static, jemalloc-enabled)
RUN git clone --branch ${ROCKS_DB_VERSION} --depth 1 https://github.com/facebook/rocksdb /tmp/rocksdb; \
    cd /tmp/rocksdb; \
    mkdir -p build && cd build; \
    cmake -G "Unix Makefiles" .. \
      -DROCKSDB_BUILD_SHARED=OFF \
      -DROCKSDB_BUILD_STATIC=ON \
      -DWITH_TESTS=OFF \
      -DWITH_TOOLS=OFF \
      -DWITH_GFLAGS=OFF \
      -DWITH_BZ2=ON \
      -DWITH_LZ4=ON \
      -DWITH_SNAPPY=ON \
      -DWITH_ZLIB=ON \
      -DWITH_ZSTD=ON \
      -DUSE_RTTI=1 \
      -DWITH_JEMALLOC=ON \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_POSITION_INDEPENDENT_CODE=ON; \
    cmake --build . -j$(nproc); \
    cmake --install . --config Release; \
    strip /usr/local/lib/librocksdb.a || true; \
    rm -rf /tmp/rocksdb

# Point CGO at the static RocksDB artifacts so the final binary does not miss runtime libs
ENV ROCKSDB_STATIC_LDFLAGS="-L/usr/local/lib -L/usr/lib -lrocksdb -lsnappy -lbz2 -lz -llz4 -lzstd -ljemalloc -lstdc++ -ldl -lpthread" \
    CGO_LDFLAGS="${ROCKSDB_STATIC_LDFLAGS}" \
    CGO_CFLAGS="-I/usr/local/include"

# Determine GOARCH and download the appropriate libraries
RUN set -eux; \
    case "${TARGETARCH}" in \
        "amd64") export GOARCH="amd64"; ARCH="x86_64";; \
        "arm64") export GOARCH="arm64"; ARCH="aarch64";; \
        *) echo "Unsupported architecture: ${TARGETARCH}"; exit 1;; \
    esac; \
    echo "Using GOARCH=${GOARCH} and ARCH=${ARCH}"; \
    wget -O /lib/libmovevm_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libmovevm_muslc.${ARCH}.a; \
    wget -O /lib/libcompiler_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libcompiler_muslc.${ARCH}.a; \
    cp /lib/libmovevm_muslc.${ARCH}.a /lib/libmovevm_muslc.a; \
    cp /lib/libcompiler_muslc.${ARCH}.a /lib/libcompiler_muslc.a

# Verify the library hashes (optional, uncomment if needed)
# RUN sha256sum /lib/libmovevm_muslc.${ARCH}.a | grep ...
# RUN sha256sum /lib/libcompiler_muslc.${ARCH}.a | grep ...

# Build the project with the specified architecture and linker flags
RUN COSMOS_BUILD_OPTIONS=rocksdb VERSION=${VERSION} COMMIT=${COMMIT} LEDGER_ENABLED=false BUILD_TAGS=muslc GOARCH=${GOARCH} LDFLAGS="-linkmode=external -extldflags \"${ROCKSDB_STATIC_LDFLAGS} -Wl,-z,muldefs -static\"" make build

FROM alpine:3.19

# install curl for the health check
RUN apk add curl

RUN addgroup minitia \
    && adduser -G minitia -D -h /minitia minitia

WORKDIR /minitia

COPY --from=go-builder /code/build/minitiad /usr/local/bin/minitiad

USER minitia

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/local/bin/minitiad", "version"]
