ARG IMAGE=alpine
ARG BASE_VERSION=latest

FROM ${IMAGE}:${BASE_VERSION}

ENV ZIG_VERSION=0.13.0
ENV ZIG_PATH=/zig/${ZIG_VERSION}/files

RUN apk add --no-cache \
        ca-certificates \
        wget

RUN apkArch="$(apk --print-arch)"; \
    case "$apkArch" in \
        x86_64) zigupURL='https://github.com/marler8997/zigup/releases/download/v2024_05_05/zigup-x86_64-linux.tar.gz' ;; \
        aarch64) zigupURL='https://github.com/marler8997/zigup/releases/download/v2024_05_05/zigup-aarch64-linux.tar.gz' ;; \
        armv7) zigupURL='https://github.com/marler8997/zigup/releases/download/v2024_05_05/zigup-arm-linux.tar.gz' ;; \
        ppc64le) zigupURL='https://github.com/marler8997/zigup/releases/download/v2024_05_05/zigup-powerpc64le-linux.tar.gz' ;; \
        riscv64) zigupURL='https://github.com/marler8997/zigup/releases/download/v2024_05_05/zigup-riscv64-linux.tar.gz' ;; \
        *) echo >&2 "unsupported architecture: $apkArch"; exit 1 ;; \
    esac; \
    wget -q "$zigupURL" && \
    tar -xzf "$(basename $zigupURL)" -C /usr/bin && \
    chmod +x /usr/bin/zigup && \
    zigup --install-dir /zig ${ZIG_VERSION} \
    && chmod -R a+w ${ZIG_PATH} \
    && rm $PWD/*.tar.gz \
    /usr/bin/zigup;

ENV PATH=/usr/bin:$PATH


RUN mkdir -p /app
WORKDIR /app



