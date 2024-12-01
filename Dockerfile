FROM node:18 AS frontend-builder
WORKDIR /app
ARG VITE_CONTACT_EMAIL
ENV VITE_CONTACT_EMAIL=${VITE_CONTACT_EMAIL}
COPY bindle-client .
RUN npm install
RUN npm run build

FROM golang:alpine AS backend-builder
WORKDIR /app
# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev
COPY bindle-server .
RUN go build -o /app/bindle ./cmd/server/main.go
RUN ls -la /app

FROM alpine:latest
# Add runtime dependencies for SQLite
RUN apk add --no-cache sqlite
WORKDIR /root/
COPY --from=backend-builder /app/bindle ./
COPY --from=frontend-builder /app/build ./static

EXPOSE 3000
CMD ["./bindle"]
