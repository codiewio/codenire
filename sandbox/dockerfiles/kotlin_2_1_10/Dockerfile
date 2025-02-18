FROM openjdk:8-jdk-alpine

RUN apk add --no-cache bash

RUN apk add --no-cache curl unzip && \
    curl -LO "https://github.com/JetBrains/kotlin/releases/download/v2.1.10/kotlin-compiler-2.1.10.zip" && \
    unzip kotlin-compiler-2.1.10.zip -d /usr/local && \
    ln -s /usr/local/kotlin-compiler-2.1.10/bin/kotlinc /usr/local/bin/kotlinc && \
    ln -s /usr/local/kotlin-compiler-2.1.10/bin/kotlin /usr/local/bin/kotlin && \
    rm kotlin-compiler-2.1.10.zip

RUN mkdir -p /app
WORKDIR /app

ENV PATH=$PATH:/usr/local/kotlinc/bin

