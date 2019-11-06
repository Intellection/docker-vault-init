FROM golang:1.10.2

# ARGs & ENVs
ARG APP_USER="app"
ENV HOME="/home/${APP_USER}"
ARG CODE_PATH="/srv/code"
ARG VAULT_INIT_VERSION
ARG VAULT_INIT_PACKAGE="vault-init-${VAULT_INIT_VERSION}-linux-amd64"

RUN mkdir -p ${CODE_PATH}
WORKDIR ${CODE_PATH}

RUN wget --progress="dot:mega" "https://github.com/Intellection/vault-init/releases/download/${VAULT_INIT_VERSION}/${VAULT_INIT_PACKAGE}.tar.gz" && \
    wget --progress="dot:mega" "https://github.com/Intellection/vault-init/releases/download/${VAULT_INIT_VERSION}/${VAULT_INIT_PACKAGE}-shasum-256.txt" && \
    sha256sum -c "${VAULT_INIT_PACKAGE}-shasum-256.txt" && \
    tar --no-same-owner -xzf "${VAULT_INIT_PACKAGE}.tar.gz" && \
    mv "/${VAULT_INIT_PACKAGE}" "/usr/local/bin/vault-init" && \
    chmod +x "/usr/local/bin/vault-init" && \
    rm -f ${VAULT_INIT_PACKAGE}*

# User
RUN groupadd -g 2019 ${APP_USER} && \
    useradd --system --create-home -u 2019 -g 2019 ${APP_USER}

COPY . .
