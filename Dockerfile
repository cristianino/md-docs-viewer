# syntax=docker/dockerfile:1
#
# Imagen de mdocs. Dos variantes:
#   docker build -t mdocs .                 completa: incluye Chromium (PDF y draw.io en vivo)
#   docker build -t mdocs:slim --target slim .   mínima (distroless): sin Chrome,
#                                           para --exports static u off
# El proyecto de documentación se monta (o copia) en /work.

# La compilación corre en la arquitectura del host y cruza a la del destino
# (sin emulación): docker buildx build --platform linux/amd64,linux/arm64 .
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w -X github.com/cristianino/mdocs/cmd/mdocs.Version=${VERSION}" \
      -o /out/mdocs .

# --- mínima: sin Chrome ---------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS slim
COPY --from=build /out/mdocs /usr/local/bin/mdocs
WORKDIR /work
ENV MDOCS_HOST=0.0.0.0 PORT=8080 MDOCS_EXPORTS=static
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD ["/usr/local/bin/mdocs", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/mdocs"]
CMD ["serve"]

# --- completa: con Chromium (por defecto) ---------------------------------
FROM debian:bookworm-slim AS full
RUN apt-get update \
 && apt-get install -y --no-install-recommends chromium fonts-liberation fonts-dejavu-core ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --create-home --uid 10001 mdocs
COPY --from=build /out/mdocs /usr/local/bin/mdocs
USER mdocs
WORKDIR /work
# Sin user namespaces el sandbox de Chrome no arranca dentro del contenedor.
ENV MDOCS_HOST=0.0.0.0 PORT=8080 MDOCS_CHROME=/usr/bin/chromium MDOCS_NO_SANDBOX=1
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD ["mdocs", "healthcheck"]
ENTRYPOINT ["mdocs"]
CMD ["serve"]
