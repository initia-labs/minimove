# Stage 1: Build the Go project
FROM golang:1.23-alpine AS go-builder

ARG TARGETARCH
ARG VERSION
ARG COMMIT

ENV LIBMOVEVM_VERSION=v1.1.1
ENV ROCKS_DB_VERSION=v10.5.1

RUN set -eux; \
    apk add --no-cache \
        ca-certificates build-base git cmake curl bash perl coreutils \
        linux-headers \
        snappy-dev snappy-static \
        zlib-dev zlib-static \
        bzip2-dev bzip2-static \
        lz4-dev lz4-static \
        zstd-dev zstd-static \
        jemalloc-dev jemalloc-static

ENV CC=gcc CXX=g++

WORKDIR /code
COPY . .

# Build static RocksDB (official INSTALL.md)
RUN git clone --branch ${ROCKS_DB_VERSION} --depth 1 https://github.com/facebook/rocksdb /tmp/rocksdb && \
    cd /tmp/rocksdb && \
    PORTABLE=1 \
    USE_RTTI=1 \
    LITE=0 \
    DISABLE_GFLAGS=1 \
    USE_JEMALLOC=1 \
    make -j$(nproc) static_lib && \
    make install-static && \
    strip /usr/local/lib/librocksdb.a || true && \
    rm -rf /tmp/rocksdb

# Static link flags for RocksDB + compression libs
ENV ROCKSDB_STATIC_LDFLAGS="-L/usr/local/lib \
    -lrocksdb -lsnappy -lbz2 -lz -llz4 -lzstd -ljemalloc \
    -lstdc++fs -lstdc++ -ldl -lpthread"

ENV CGO_LDFLAGS="${ROCKSDB_STATIC_LDFLAGS}"
ENV CGO_CFLAGS="-I/usr/local/include"

# Download MoveVM static libraries depending on platform
RUN set -eux; \
    case "${TARGETARCH}" in \
        "amd64") ARCH="x86_64"; export GOARCH="amd64";; \
        "arm64") ARCH="aarch64"; export GOARCH="arm64";; \
        *) echo "Unsupported arch: ${TARGETARCH}"; exit 1;; \
    esac; \
    wget -O /lib/libmovevm_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libmovevm_muslc.${ARCH}.a; \
    wget -O /lib/libcompiler_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libcompiler_muslc.${ARCH}.a; \
    cp /lib/libmovevm_muslc.${ARCH}.a /lib/libmovevm_muslc.a; \
    cp /lib/libcompiler_muslc.${ARCH}.a /lib/libcompiler_muslc.a

# Build static binary
RUN COSMOS_BUILD_OPTIONS=rocksdb \
    VERSION=${VERSION} \
    COMMIT=${COMMIT} \
    BUILD_TAGS=muslc \
    LEDGER_ENABLED=false \
    GOARCH=${GOARCH} \
    LDFLAGS="-linkmode=external -extldflags '${ROCKSDB_STATIC_LDFLAGS} -static -Wl,-z,muldefs'" \
    make build

# ────────────────────────────────
# Final minimal runtime image
# ────────────────────────────────
FROM alpine:3.19

RUN addgroup minitia && adduser -G minitia -D -h /minitia minitia
WORKDIR /minitia

# Optional: curl for health check
RUN apk add --no-cache curl

COPY --from=go-builder /code/build/minitiad /usr/local/bin/minitiad

USER minitia

EXPOSE 1317 9090 26656 26657

CMD ["/usr/local/bin/minitiad", "version"]
