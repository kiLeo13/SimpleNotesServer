FROM golang:1.25 AS builder
LABEL maintainer="Leonardo"

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0
WORKDIR /src

# Defines it as production environment
ENV GO_ENV=production

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} \
    go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/api


### Final stage (tiny runtime)
FROM gcr.io/distroless/static:nonroot

EXPOSE 7070

COPY --from=builder /app/server /app/server

USER nonroot
ENTRYPOINT ["/app/server"]