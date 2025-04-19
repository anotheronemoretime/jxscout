# Étape 1: Image de base avec Go et Bun
FROM oven/bun:latest AS base

# Installation de Go 1.23.2
RUN apt-get update && apt-get install -y \
    wget \
    git \
    build-essential \ 
    && rm -rf /var/lib/apt/lists/*

RUN wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz \
    && rm go1.23.2.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

# Étape 2: Construction de l'application
WORKDIR /app

# Copie des fichiers du projet
COPY . .

# Installation des dépendances et compilation
RUN make install
RUN make build

# Étape 3: Configuration finale
# Création du dossier pour le volume
RUN mkdir -p /home/bun/jxscout/default

# Variables d'environnement pour la configuration
ENV JXSCOUT_HOSTNAME="0.0.0.0"
ENV JXSCOUT_PROXY_URL=""
ENV JXSCOUT_DEBUG="false"

# Exposition du port
EXPOSE 3333

# Création d'un script d'entrée
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

# Utilisation du script comme point d'entrée
ENTRYPOINT ["/app/entrypoint.sh"]