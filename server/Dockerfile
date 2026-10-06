FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/nora-theme-store ./cmd/themestore

FROM alpine:3.22
RUN addgroup -S nora && adduser -S -G nora nora
WORKDIR /app
COPY --from=build /out/nora-theme-store /app/nora-theme-store
RUN mkdir -p /data && chown -R nora:nora /data /app
USER nora
ENV NORA_DEPLOY_MODE=lan LISTEN_ADDR=0.0.0.0:8090 DATA_DIR=/data DATABASE_PATH=/data/themes.db
EXPOSE 8090
VOLUME ["/data"]
ENTRYPOINT ["/app/nora-theme-store"]
