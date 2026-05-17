ARG BUILDER_IMAGE=docker.m.daocloud.io/eclipse-temurin:21-jdk
ARG RUNTIME_IMAGE=docker.m.daocloud.io/eclipse-temurin:21-jdk

FROM ${BUILDER_IMAGE} AS build
WORKDIR /src
COPY src ./src
RUN mkdir -p /out/classes \
    && javac --release 17 -d /out/classes $(find src/main/java -name '*.java') \
    && jar --create --file /out/jdk17-http-demo.jar --main-class com.ebpfjava.examples.httpdemo.DemoHttpService -C /out/classes .

FROM ${RUNTIME_IMAGE}
WORKDIR /app
COPY --from=build /out/jdk17-http-demo.jar /app/jdk17-http-demo.jar
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-jar", "/app/jdk17-http-demo.jar"]
