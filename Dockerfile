FROM oven/bun:latest AS base

RUN apt-get update && apt-get install -y \
    wget \
    git \
    build-essential \ 
    && rm -rf /var/lib/apt/lists/*

RUN wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz \
    && rm go1.23.2.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

COPY . .

RUN make install
RUN make clean
RUN make build

RUN apt-get update && apt-get install -y gcc libsqlite3-dev
RUN go build -o ./dist/jxscout ./cmd/jxscout

RUN mkdir -p /home/bun/jxscout/default

ENV JXSCOUT_HOSTNAME="0.0.0.0"
ENV JXSCOUT_PROXY_URL=""
ENV JXSCOUT_DEBUG="false"

EXPOSE 3333

RUN echo '#!/bin/sh\n\
ARGS=""\n\
\n\
if [ -n "$JXSCOUT_HOSTNAME" ]; then\n\
  ARGS="$ARGS -hostname $JXSCOUT_HOSTNAME"\n\
fi\n\
\n\
if [ -n "$JXSCOUT_PROXY_URL" ]; then\n\
  ARGS="$ARGS -proxy-url $JXSCOUT_PROXY_URL"\n\
fi\n\
\n\
if [ "$JXSCOUT_DEBUG" = "true" ]; then\n\
  ARGS="$ARGS -debug"\n\
fi\n\
\n\
exec ./dist/jxscout $ARGS "$@"\n' > /app/entrypoint.sh \
    && chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]