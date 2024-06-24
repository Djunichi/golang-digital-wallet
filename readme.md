# Docker Compose Setup for NATS, PostgreSQL, User Service, Transactions Service, and pgAdmin

This repository contains a Docker Compose setup for running a NATS server, a PostgreSQL database, a user service, a transactions service, and pgAdmin. Follow the instructions below to get started.

## Prerequisites

- Docker installed on your machine
- Docker Compose installed on your machine

## Services

### NATS
[NATS](https://nats.io/) is a simple, secure, and high-performance messaging system for cloud-native applications, IoT messaging, and microservices architectures.

- **Ports**: 4222

### PostgreSQL
[PostgreSQL](https://www.postgresql.org/) is a powerful, open-source object-relational database system.

- **Ports**: 5432
- **Healthcheck**:
  - Test: `pg_isready -U testUser -d testDb`
  - Interval: 10s
  - Timeout: 5s
  - Retries: 5

### User Service
This service connects to NATS and PostgreSQL to manage user data.

- **Ports**: 8080
- **Depends On**:
  - NATS (Service started)
  - PostgreSQL (Service healthy)

### Transactions Service
This service connects to NATS and PostgreSQL to manage transaction data.

- **Ports**: 8081
- **Depends On**:
  - NATS (Service started)
  - PostgreSQL (Service healthy)

### pgAdmin
[pgAdmin](https://www.pgadmin.org/) is a web-based administration tool for PostgreSQL.

- **Ports**: 5050 (Mapped to 80 in the container)
- **Environment Variables**:
  - `PGADMIN_DEFAULT_EMAIL`: admin@example.com
  - `PGADMIN_DEFAULT_PASSWORD`: admin
- **Depends On**: PostgreSQL

## Networks

- `backend`: A custom network for the services to communicate with each other.

## Getting Started

1. Clone the repository:
    ```sh
    git clone https://github.com/Djunichi/golang-digital-wallet.git
    cd golang-digital-wallet
    ```

2. Ensure you have the necessary files in place:
  - `init.sql` for initializing the PostgreSQL database.
  - `postgresql.conf` for PostgreSQL configuration.

3. Run the Docker Compose setup:
    ```sh
    docker-compose up -d
    ```

4. Access the services:
  - NATS: `localhost:4222`
  - PostgreSQL: `localhost:5432`
  - User Service: `localhost:8080/swagger/index.html`
  - Transactions Service: `localhost:8081/swagger/index.html`
  - pgAdmin: `localhost:5050`

## Stopping the Setup

To stop the Docker Compose setup, run:
```sh
docker-compose down
```
## Troubleshooting
If you encounter issues with starting the services, check the Docker Compose logs for errors:
```sh
docker-compose logs
```