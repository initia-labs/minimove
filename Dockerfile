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

WORKDIR /code
COPY . /code/

# Install rocksdb
RUN git clone --branch ${ROCKS_DB_VERSION} --depth 1 https://github.com/facebook/rocksdb; cd rocksdb; make -j$(nproc) static_lib; make install-static

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
