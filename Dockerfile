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
    fonts-baekmuk \
    fonts-farsiweb \
    fonts-hosny-amiri \
    fonts-nafees \
    fonts-noto \
    fonts-sil-andika \
    fonts-sil-charis \
    fonts-sil-doulos \
    fonts-sil-ezra \
    fonts-sil-galatia \
    fonts-sil-gentium \
    fonts-sil-padauk \
    fonts-sil-scheherazade \
    git \
    imagemagick \
    k2pdfopt \
    pdftk-java \
    poppler-utils \
    xmlstarlet \
    xz-utils \
    yq \
    && rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

RUN curl -L https://github.com/go-task/task/releases/download/v3.45.5/task_linux_amd64.deb --output /tmp/task_linux_amd64.deb \
    && curl -L http://ftp.pl.debian.org/debian/pool/main/c/culmus/fonts-culmus_0.140-3_all.deb --output /tmp/fonts-culmus_0.140-3_all.deb \
    && curl -L http://ftp.pl.debian.org/debian/pool/main/c/culmus-fancy/fonts-culmus-fancy_0.0.20240129.1_all.deb --output /tmp/fonts-culmus-fancy_0.0.20240129.1_all.deb \
    && dpkg --install /tmp/*.deb \
    && rm -rf /tmp/* \
    && pip install --upgrade pip \
    && pip install anki

    
    # && mkdir -p /usr/local/share/fonts/truetype/simsun \
    # && curl -L https://github.com/wuhongyi/fonts/raw/refs/heads/master/simkai.ttf --output /usr/local/share/fonts/truetype/simsun/simkai.ttf \
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simfang.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simhei.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simli.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simsun.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simyou.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/%E5%AE%8B%E4%BD%93-%E7%B2%97%E4%BD%93.ttf --output 宋体-粗体.ttf
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/simsun.ttc
    # https://github.com/wuhongyi/fonts/raw/refs/heads/master/uming.ttc
    # && fc-cache -fv \

WORKDIR /workspace

COPY dist/*-cli /usr/bin
COPY cfg/linux.yml /root/.config/cli-tools/config

ENV DEBIAN_FRONTEND=dialog
