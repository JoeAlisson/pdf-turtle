FROM golang as build-service
WORKDIR /build
COPY . .
RUN go build -o pdf-turtle


FROM chromedp/headless-shell:stable as runtime
RUN apt-get -y update
RUN apt-get -y install ca-certificates fonts-open-sans fonts-roboto fonts-noto-color-emoji
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build-service /build/pdf-turtle /app/pdf-turtle

ENV LANG en-US.UTF-8
ENV LOG_LEVEL_DEBUG false
ENV LOG_JSON_OUTPUT false
ENV WORKER_INSTANCES 40
ENV PORT 8000

EXPOSE ${PORT}

ENTRYPOINT ["/app/pdf-turtle"]