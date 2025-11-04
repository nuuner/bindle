FROM docker.io/node:20-alpine AS frontend-builder
WORKDIR /app
COPY bindle-client/package*.json ./
# Install all dependencies including devDependencies needed for build
RUN npm ci || npm install
COPY bindle-client .
ARG VITE_CONTACT_EMAIL
ENV VITE_CONTACT_EMAIL=$VITE_CONTACT_EMAIL
RUN npm run build

FROM docker.io/golang:alpine AS backend-builder
WORKDIR /app
# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev
COPY bindle-server .
RUN go build -o /app/bindle ./cmd/server/main.go

FROM docker.io/alpine:latest
# Add runtime dependencies for SQLite
RUN apk add --no-cache sqlite
WORKDIR /app
COPY --from=backend-builder /app/bindle ./
COPY --from=frontend-builder /app/build ./static

EXPOSE 3000
CMD ["./bindle"]
