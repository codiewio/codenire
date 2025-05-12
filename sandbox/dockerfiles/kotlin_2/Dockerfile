FROM gradle:jdk22

RUN mkdir -p /gradle-deps && mkdir -p /provision

COPY build.gradle.kts /gradle-deps/build.gradle.kts
COPY gradle.properties /gradle-deps/gradle.properties
RUN #cd /gradle-deps && gradle build --refresh-dependencies


COPY app.build.gradle.kts /provision/build.gradle.kts

RUN mkdir -p /app
WORKDIR /app

#
COPY Main.kt ./src/main/kotlin/Main.kt
RUN cp -r /provision/. /app/
RUN gradle build --refresh-dependencies
#

