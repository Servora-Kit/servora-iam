FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG SERVICE_NAME=iam
ARG VERSION=dev

RUN apk add --no-cache git

WORKDIR /src

COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY api/gen/go.mod api/gen/go.sum ./api/gen/
COPY app/iam/service/go.mod app/iam/service/go.sum ./app/iam/service/
COPY app/sayhello/service/go.mod app/sayhello/service/go.sum ./app/sayhello/service/

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Name=${SERVICE_NAME}.service" \
    -o /src/bin/${SERVICE_NAME} ./app/${SERVICE_NAME}/service/cmd/server

FROM alpine:3.19

ARG SERVICE_NAME=iam

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /src/bin/${SERVICE_NAME} /app/${SERVICE_NAME}

VOLUME /app/configs

ENV TZ=Asia/Shanghai
ENV SERVICE_NAME=${SERVICE_NAME}

CMD ["/bin/sh", "-c", "/app/${SERVICE_NAME} -conf /app/configs/"]
