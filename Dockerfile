# Stage 1: Build the Go project
FROM initia/go-rocksdb-builder:0001-alpine AS go-builder

ARG TARGETARCH
ARG VERSION
ARG COMMIT

ENV LIBMOVEVM_VERSION=v1.1.1

WORKDIR /code
COPY . .

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
    LDFLAGS="-linkmode=external -extldflags \"-static -Wl,-z,muldefs\"" \
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
