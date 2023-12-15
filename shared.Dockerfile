FROM golang:1.19.1-buster AS go-builder

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editting in the image
RUN apt install -y vim

RUN git config --global url."https://x-access-token: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@github.com/".insteadOf "https://github.com/"
RUN go env -w GOPRIVATE=github.com/initia-labs/*


WORKDIR /code
COPY . /code/

RUN LEDGER_ENABLED=false make build

RUN cp /go/pkg/mod/github.com/initia\-labs/initiavm@v*/api/libinitia.`uname -m`.so /lib/libinitia.so

FROM ubuntu:20.04

WORKDIR /root

COPY --from=go-builder /code/build/minitiad /usr/local/bin/minitiad
COPY --from=go-builder /lib/libinitia.so /lib/libinitia.so

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/local/bin/minitiad", "version"]
