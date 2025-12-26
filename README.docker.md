# Docker Setup Guide

This project includes a Docker Compose configuration for easy development environment setup.

## Services Included

- **PostgreSQL 16** - Main database (port 5432)
- **Redis 7** - Cache and session storage (port 6379)
- **pgAdmin** - PostgreSQL management UI (port 5050)
- **Redis Commander** - Redis management UI (port 8081)
- **MailHog** - Email testing tool (SMTP: 1025, Web UI: 8025)

## Quick Start

### 1. Start all services
```bash
docker-compose up -d
```

### 2. Stop all services
```bash
docker-compose down
```

### 3. Stop and remove volumes (clean start)
```bash
docker-compose down -v
```

### 4. View logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f postgres
docker-compose logs -f redis
```

## Service Access

### PostgreSQL
- **Host**: localhost
- **Port**: 5432
- **Database**: upbot_db
- **User**: upbot
- **Password**: upbot_password
- **Connection String**: `postgresql://upbot:upbot_password@localhost:5432/upbot_db?sslmode=disable`

### Redis
- **Host**: localhost
- **Port**: 6379
- **Connection String**: `localhost:6379`

### pgAdmin (PostgreSQL GUI)
- **URL**: http://localhost:5050
- **Email**: admin@upbot.com
- **Password**: admin

To connect to PostgreSQL in pgAdmin:
1. Right-click "Servers" → "Register" → "Server"
2. General Tab: Name = "Upbot Local"
3. Connection Tab:
   - Host: postgres (or use host.docker.internal if pgAdmin is not in the same network)
   - Port: 5432
   - Database: upbot_db
   - Username: upbot
   - Password: upbot_password

### Redis Commander (Redis GUI)
- **URL**: http://localhost:8081

### MailHog (Email Testing)
- **SMTP Server**: localhost:1025
- **Web UI**: http://localhost:8025

Use MailHog to test email functionality without sending real emails. All emails sent to the SMTP server will appear in the web UI.

## Environment Variables

Copy `.env.example` to `.env` and update the values:

```bash
cp .env.example .env
```

The default `.env.example` is already configured to work with the Docker services.

## Running the Go Application

With Docker services running, start your Go application:

```bash
go run main.go
```

Or:

```bash
go run cmd/server/main.go
```

## Useful Commands

### Check service health
```bash
docker-compose ps
```

### Restart a specific service
```bash
docker-compose restart postgres
docker-compose restart redis
```

### Execute commands in containers
```bash
# PostgreSQL shell
docker-compose exec postgres psql -U upbot -d upbot_db

# Redis CLI
docker-compose exec redis redis-cli
```

### Backup PostgreSQL database
```bash
docker-compose exec postgres pg_dump -U upbot upbot_db > backup.sql
```

### Restore PostgreSQL database
```bash
docker-compose exec -T postgres psql -U upbot upbot_db < backup.sql
```

## Troubleshooting

### Port already in use
If you get a "port already in use" error, either:
1. Stop the conflicting service on your machine
2. Change the port mapping in `docker-compose.yml` (e.g., `"5433:5432"` for PostgreSQL)

### Cannot connect to database
1. Ensure containers are running: `docker-compose ps`
2. Check logs: `docker-compose logs postgres`
3. Verify connection string in `.env` matches the service configuration

### Reset everything
```bash
docker-compose down -v
docker-compose up -d
```

## Production Notes

⚠️ **This configuration is for development only!**

For production:
- Use strong passwords
- Enable SSL/TLS for PostgreSQL
- Use proper secrets management
- Configure proper backup strategies
- Remove development tools (pgAdmin, Redis Commander, MailHog)
- Use managed database services when possible
