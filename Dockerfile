FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY bindle-client .
ARG VITE_CONTACT_EMAIL
ENV VITE_CONTACT_EMAIL=$VITE_CONTACT_EMAIL
# Clean npm cache and install with better compatibility
RUN npm cache clean --force && \
    rm -rf node_modules package-lock.json && \
    npm install --legacy-peer-deps && \
    npm rebuild && \
    npm run build

FROM golang:alpine AS backend-builder
WORKDIR /app
# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev
COPY bindle-server .
RUN go build -o /app/bindle ./cmd/server/main.go

FROM alpine:latest
# Add runtime dependencies for SQLite
RUN apk add --no-cache sqlite
WORKDIR /app
COPY --from=backend-builder /app/bindle ./
COPY --from=frontend-builder /app/build ./static

EXPOSE 3000
CMD ["./bindle"]
