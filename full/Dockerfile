# build apron-node
FROM paritytech/ci-linux:production as builder

ARG GIT_REPO="https://github.com/Apron-Network/apron-node.git"
ARG GIT_BRANCH="sub3.0_dev"

RUN git clone -b ${GIT_BRANCH} --recursive ${GIT_REPO}
WORKDIR /builds/apron-node

RUN cargo build
RUN cp target/debug/apron-node /apron-node

# build polkadot and apron  frontend
FROM ubuntu:18.04 as frontend_builder

ARG FRONT_REPO="https://github.com/Apron-Network/apps.git"
ARG FRONT_BRANCH="master"

RUN apt-get update && apt-get install --no-install-recommends -y git gnupg curl ca-certificates

# install nodejs
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash -
RUN apt-get install --no-install-recommends -y nodejs && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*
RUN npm install yarn -g

RUN mkdir /builds && cd /builds && git clone -b ${FRONT_BRANCH} ${FRONT_REPO}
WORKDIR /builds/apps

RUN YARN_CHECKSUM_BEHAVIOR=update yarn && yarn build:www

# build gateway
FROM golang:1.16-buster AS gateway_builder
ARG GW_REPO="https://github.com/Apron-Network/apron-gateway-p2p.git"
ARG GW_BRANCH="master"

WORKDIR /src
RUN git clone -b ${GW_BRANCH} --recursive ${GW_REPO}
RUN cd /src/apron-gateway-p2p  && make build

FROM ubuntu
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install curl ca-certificates npm nodejs -y
RUN npm install -g serve
COPY --from=builder /apron-node /

COPY --from=frontend_builder /builds/apps/packages/apps /apps

COPY --from=gateway_builder /src/apron-gateway-p2p/gateway /

EXPOSE 30333 9933 9944 8080 8082 2145

