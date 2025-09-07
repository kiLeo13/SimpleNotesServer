FROM golang:1.25 AS builder
LABEL maintainer="Leonardo"

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0
WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} \
    go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/api


### Final stage (tiny runtime)
FROM gcr.io/distroless/static:nonroot

EXPOSE 7070

COPY --from=builder /app/server /app/server

USER nonroot
ENTRYPOINT ["/app/server"]