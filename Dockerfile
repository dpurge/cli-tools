FROM python:3.14-slim-trixie

ARG VERSION=${VERSION}
ARG BRANCH=${BRANCH}
ARG COMMIT=${COMMIT}

ENV APP_VERSION=${VERSION}
ENV GIT_BRANCH=${BRANCH}
ENV GIT_COMMIT=${COMMIT}

ENV DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC

RUN apt update && apt install -y --no-install-recommends \
    calibre \
    djvulibre-bin \
    ffmpeg \
    fonts-arphic-ukai \
    fonts-arphic-uming \
    fonts-hosny-amiri \
    fonts-noto \
    git \
    imagemagick \
    k2pdfopt \
    pdftk-java \
    poppler-utils \
    xmlstarlet \
    xz-utils \
    yq \
    && apt clean
    # && rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

RUN curl -L https://github.com/go-task/task/releases/download/v3.45.5/task_linux_amd64.deb --output /tmp/task_linux_amd64.deb \
    && dpkg --install /tmp/*.deb \
    && rm -rf /tmp/*

WORKDIR /workspace

COPY dist/*-cli /usr/bin
COPY cfg/linux.yml /root/.config/cli-tools/config

ENV DEBIAN_FRONTEND=dialog
