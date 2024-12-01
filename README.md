# Bindle

Bindle is a modern file sharing platform built with Go and Svelte. It provides a simple, secure way to upload and share files through a clean web interface.

## Features

- Fast and lightweight
- Account-based file management
- Support for multiple storage backends (Local filesystem & S3)
- File preview support for images, videos, audio, and text files
- Responsive design
- Drag & drop file uploads
- Storage quota management

## Tech Stack

- **Frontend**: Svelte 5, TailwindCSS, Carbon Components
- **Backend**: Go, Fiber
- **Storage**: Local filesystem or S3-compatible storage
- **Database**: SQLite

## Getting Started

1. Clone the repository:
```bash
git clone https://github.com/nuuner/bindle.git
cd bindle
```

2. Create a `.env` file in the `bindle-server` directory:
```env
ENVIRONMENT=development
FILESYSTEM_PATH=./files
UPLOAD_LIMIT_BYTES=104857600  # 100MB
```

3. Start the development environment:
```bash
docker compose up --build
```

The application will be available at `http://localhost:3000`.

## Development

### Frontend

```bash
cd bindle-client
npm install
npm run dev
```

### Backend

```bash
cd bindle-server
go run cmd/server/main.go
```

## Deployment

The project includes a Docker configuration for easy deployment. Build and run using:

```bash
docker compose up --build -d
```

## License

GPLv3

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.