FROM node:22-alpine AS frontend
WORKDIR /src/static
COPY static/package.json static/package-lock.json ./
RUN npm ci --no-fund
COPY static/ ./
RUN npm run build

FROM golang:1.27.4-alpine AS backend
WORKDIR /src
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY handler/ ./handler/
COPY internal/ ./internal/
COPY logic/ ./logic/
COPY misc/ ./misc/
COPY repo/ ./repo/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gomusic .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /out/gomusic ./gomusic
COPY --from=frontend /src/static/dist ./static/dist
USER 65532:65532
ENV GIN_MODE=release GOMUSIC_ADDR=:8081
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD wget -q -O /dev/null http://127.0.0.1:8081/healthz || exit 1
ENTRYPOINT ["./gomusic"]
