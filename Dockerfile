FROM golang:alpine3.19 AS build-service
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY config/ config/
COPY loopback/ loopback/
COPY static-files/ static-files/
COPY serverutils/ serverutils/
COPY utils/ utils/
COPY server/ server/
COPY services/ services/
COPY models/ models/
COPY main.go .

RUN go build -o pdf-turtle


FROM node:21.7.1-alpine3.19 AS build-playground
WORKDIR /app
COPY .playground/package*.json ./
RUN npm ci --ignore-scripts

COPY .playground/ ./
RUN npm run build


FROM chromedp/headless-shell:124.0.6367.60 AS runtime
WORKDIR /app

RUN apt update && \
    apt install -y ca-certificates fonts-noto-color-emoji fonts-open-sans fonts-roboto && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

ENV LANG en-US.UTF-8
ENV LOG_LEVEL_DEBUG false
ENV LOG_JSON_OUTPUT false
ENV WORKER_INSTANCES 40
ENV PORT 8000
ENV SERVE_PLAYGROUND true
ENV NO_SANDBOX true

EXPOSE ${PORT}

RUN useradd -u 64198 app
USER app

COPY --from=build-service /build/pdf-turtle /app/pdf-turtle
COPY --from=build-playground /app/dist /app/static-files/extern/playground

ENTRYPOINT ["/app/pdf-turtle"]
