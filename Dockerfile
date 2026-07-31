# syntax=docker/dockerfile:1

# ---- Stage 1: fetch the static Typst binary ----
# Typst ships a fully static (musl, static-pie) x86_64 Linux binary, so it runs
# on this glibc-based Debian runtime unchanged. Fetched in an isolated stage so
# curl/xz-utils used only for the download don't linger in the final image.
FROM debian:trixie-slim AS typst
ARG TYPST_VERSION=0.15.1
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl xz-utils \
    && curl -fsSL -o /tmp/typst.tar.xz \
        "https://github.com/typst/typst/releases/download/v${TYPST_VERSION}/typst-x86_64-unknown-linux-musl.tar.xz" \
    && tar -xJf /tmp/typst.tar.xz -C /tmp \
    && install -m 0755 /tmp/typst-x86_64-unknown-linux-musl/typst /usr/local/bin/typst \
    && rm -rf /tmp/*

# ---- Stage 2: runtime ----
FROM debian:trixie-slim

ARG VERSION=${VERSION}
ARG BRANCH=${BRANCH}
ARG COMMIT=${COMMIT}

ENV APP_VERSION=${VERSION}
ENV GIT_BRANCH=${BRANCH}
ENV GIT_COMMIT=${COMMIT}

ENV DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    fontconfig \
    djvulibre-bin \
    ffmpeg \
    fonts-arphic-ukai \
    fonts-arphic-uming \
    fonts-baekmuk \
    fonts-farsiweb \
    fonts-hosny-amiri \
    fonts-nafees \
    fonts-noto \
    fonts-noto-cjk \
    fonts-noto-extra \
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
    && rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

# mikefarah yq (Go, jq-style) installed from GitHub — NOT the Debian `yq`
# package, which is python-yq (a jq wrapper with different syntax and quoted
# output).
RUN curl -fsSL https://github.com/go-task/task/releases/download/v3.45.5/task_linux_amd64.deb --output /tmp/task_linux_amd64.deb \
    && curl -fsSL https://github.com/mikefarah/yq/releases/download/v4.53.3/yq_linux_amd64 --output /usr/local/bin/yq \
    && chmod +x /usr/local/bin/yq \
    && curl -fsSL http://ftp.pl.debian.org/debian/pool/main/c/culmus/fonts-culmus_0.140-3_all.deb --output /tmp/fonts-culmus_0.140-3_all.deb \
    && curl -fsSL http://ftp.pl.debian.org/debian/pool/main/c/culmus-fancy/fonts-culmus-fancy_0.0.20240129.1_all.deb --output /tmp/fonts-culmus-fancy_0.0.20240129.1_all.deb \
    && dpkg --install /tmp/*.deb \
    && rm -rf /tmp/* \
    && fc-cache -f

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

# Typst binary from the fetch stage above.
COPY --from=typst /usr/local/bin/typst /usr/local/bin/typst

WORKDIR /workspace

COPY dist/*-cli /usr/bin
COPY cfg/linux.yml /root/.config/cli-tools/config

ENV DEBIAN_FRONTEND=dialog
