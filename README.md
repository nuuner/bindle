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
- Admin panel for file and user management

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
# local filesystem
FILESYSTEM_PATH=./files

UPLOAD_LIMIT_MB_PER_DAY=1000

# Admin password for /admin panel (optional)
ADMIN_PASSWORD=your_secure_password_here

# Password that lifts the daily upload limit (optional)
UNLOCK_PASSWORD=your_unlock_password_here
```

or

```env
# S3
S3_BUCKET=my-bucket
S3_KEY_ID=001a2b3c4d5e6f7g8h9i0j
S3_APP_KEY=K001AbCdEfGhIjKlMnOpQrStUvWxYz
S3_REGION=us-east-1
S3_ENDPOINT=https://s3.us-east-1.amazonaws.com

UPLOAD_LIMIT_MB_PER_DAY=1000

# Admin password for /admin panel (optional)
ADMIN_PASSWORD=your_secure_password_here

# Password that lifts the daily upload limit (optional)
UNLOCK_PASSWORD=your_unlock_password_here
```

3. Also create a `.env` file in the `bindle-client` directory:

```env
VITE_CONTACT_EMAIL=test@example.com
```

4. Start the development environment:
```bash
docker compose up --build
```

The application will be available at `http://localhost:3001`.

## Running behind a reverse proxy

Rate limits and daily upload quotas are keyed on the client IP. Behind a proxy the
server sees the proxy's address on every request, which puts every user into a single
rate limit and a single shared quota. To get the real client IP, set both:

```env
TRUSTED_PROXIES=172.17.0.1,10.0.0.0/8
PROXY_HEADER=X-Real-IP
```

The header is only trusted for requests arriving from an address in `TRUSTED_PROXIES`;
anything else falls back to the socket address. Both must be set together, and the
server refuses to start with only one of them — a trusted header without a proxy
allowlist would let any client spoof its IP and shed both limits.

Your proxy must **overwrite** the header rather than append to a client-supplied value.
Check which header yours sets and whether it replaces or appends — some proxies default
to appending to `X-Forwarded-For`, which leaves the first entry attacker-controlled.

If the proxy reaches the container over a container network, trust that network's
subnet rather than the proxy's address, since container addresses change on restart.

Make sure the container's port is not also published on the host. A published port is a
second route to the app that does not pass through the proxy, and requests arriving that
way can still carry a peer address inside the trusted range — which would let a client
set the header itself and shed both the rate limit and the upload quota.

## Unlocking the daily upload limit

Everyone shares the same `UPLOAD_LIMIT_MB_PER_DAY` allowance, keyed on their IP. Set an
unlock password to give trusted people a way around it:

```env
UNLOCK_PASSWORD=your_unlock_password_here
```

They then pick **Unlock limits** from the account menu and enter the password. The server
answers with a signed, HttpOnly cookie that is valid for 30 days, and any request carrying
it skips the daily quota entirely — `MAX_FILE_SIZE_MB` still applies. The same dialog locks
the browser again, and so does clearing cookies.

Without `UNLOCK_PASSWORD` set there is no unlock: the menu option is hidden and no cookie
is honoured. Changing the password invalidates every cookie already handed out, since the
cookie is signed with the password itself. Guesses go through the same rate limit as the
other sensitive routes, but this is one shared secret for everyone who has it — treat it
like the admin password rather than a per-user login.

## Admin Panel

Bindle includes an admin panel for managing users and files. To enable it:

1. Add an admin password to your `.env` file in `bindle-server`:

```env
ADMIN_PASSWORD=your_secure_password_here
```

2. Restart the server and navigate to `/admin`

3. Enter your admin password when prompted

### Admin Features

- View all users with statistics (file count, storage usage, last login, IP addresses)
- View all files in the system with owner information
- Delete individual files
- Delete all files for a specific user
- Delete all files in the system (nuclear option)

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