# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Geth in a stock Go builder container
FROM ubuntu as builder

RUN apt-get update && apt-get install -y wget git make build-essential

RUN wget https://go.dev/dl/go1.17.7.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.7.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

ENV GO111MODULE=on
ENV GOPATH=""

RUN git clone https://github.com/gnc-project/galaxynetwork.git && \
        cd galaxynetwork && \
        go run build/ci.go install ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM ubuntu:22.04

COPY --from=builder /galaxynetwork/build/bin/geth /usr/local/bin/
#RUN pwd && ls -l /data/geth
EXPOSE 8545 8546 30303 30303/udp

CMD ["geth"]
# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"

