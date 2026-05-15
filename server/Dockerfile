# syntax=docker/dockerfile:1.7

# ---------- builder ----------
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build both binaries
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/adserver ./cmd/adserver \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---------- runtime (adserver) ----------
FROM gcr.io/distroless/static-debian12:nonroot AS adserver

COPY --from=builder /out/adserver /adserver
COPY --from=builder /out/worker /worker

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/adserver"]
