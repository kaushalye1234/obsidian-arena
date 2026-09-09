# Local Development

```bash
cp .env.example .env
docker compose up --build
```

Stop with `docker compose down`. Use `docker compose down -v` only when intentionally deleting local PostgreSQL data.

For production, start with one container and managed PostgreSQL. Terminate TLS at the load balancer. Add Redis only when horizontal scaling requires it.
