FROM golang:1.10.2
# ARGs & ENVs
ARG APP_USER="app"
ENV HOME="/home/${APP_USER}"
ARG CODE_PATH="/srv/code"
RUN mkdir -p ${CODE_PATH}
WORKDIR ${CODE_PATH}

# User
RUN groupadd -g 2019 ${APP_USER} && \
    useradd --system --create-home -u 2019 -g 2019 ${APP_USER}

COPY . .
