FROM golang:alpine3.19 AS build-service
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o pdf-turtle


FROM node:21.7.1-alpine3.19 AS build-playground
WORKDIR /app
COPY .pdf-turtle-playground/package*.json ./
RUN npm ci --ignore-scripts

COPY .pdf-turtle-playground/ ./
RUN npm run build


FROM chromedp/headless-shell:116.0.5845.14 AS runtime
WORKDIR /app

RUN apt update && apt install -y ca-certificates fonts-open-sans fonts-roboto fonts-noto-color-emoji && apt clean
RUN rm -rf /var/lib/apt/lists/*

ENV LANG en-US.UTF-8
ENV LOG_LEVEL_DEBUG false
ENV LOG_JSON_OUTPUT false
ENV WORKER_INSTANCES 40
ENV PORT 8000
ENV SERVE_PLAYGROUND true

EXPOSE ${PORT}

COPY --from=build-service /build/pdf-turtle /app/pdf-turtle
COPY --from=build-playground /app/dist /app/static-files/extern/playground

ENTRYPOINT ["/app/pdf-turtle"]
