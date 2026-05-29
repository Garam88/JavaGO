# go-commerce-api

`examples/go-commerce-api`는 교재의 주문/상품 예제를 실행 가능한 형태로 구현한 API입니다.

구성:

- HTTP API: `net/http`
- DB: PostgreSQL (`database/sql` + pgx stdlib)
- Cache: Redis cache-aside
- Message: NATS JetStream
- Event consistency: Postgres outbox pattern

## Docker Compose 실행

프로젝트 루트(`go-book`)에서 API + Postgres + Redis + NATS를 함께 실행합니다.

```bash
docker compose -f examples/docker-compose.yml up --build
```

기본 공개 포트:

- API: `8080`
- Postgres: `5432`
- Redis: `6379`
- NATS: `4222`
- NATS monitoring: `8222`

포트 충돌 시:

```bash
API_PORT=18080 POSTGRES_PORT=15432 REDIS_PORT=16379 NATS_PORT=14222 \
  docker compose -f examples/docker-compose.yml up --build
```

중지:

```bash
docker compose -f examples/docker-compose.yml down
```

데이터 볼륨까지 제거:

```bash
docker compose -f examples/docker-compose.yml down -v
```

## 로컬 API 실행

의존성만 Compose로 띄운 뒤 로컬에서 API를 실행할 수 있습니다.

```bash
docker compose -f ../docker-compose.yml up -d postgres redis nats

DB_DSN='postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable' \
REDIS_ADDR='localhost:6379' \
NATS_URL='nats://localhost:4222' \
go run ./cmd/api
```

## 기본 API

- `GET /healthz` 호환용 liveness
- `GET /livez`
- `GET /readyz`
- `GET /items`
- `GET /items/{id}`
- `POST /orders`
- `GET /orders`
- `GET /orders/{id}`

### 상품 조회

```bash
curl http://localhost:8080/items
```

### 주문 생성

초기 migration은 `sku-1`, `sku-2` 상품을 seed로 넣습니다.

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"u-1","item_id":"sku-1","quantity":2}'
```

주문 생성은 Postgres 트랜잭션 안에서 재고 차감, 주문 저장, outbox 이벤트 생성을 함께 처리합니다. 별도 outbox publisher가 `order.created` 이벤트를 NATS JetStream으로 발행하고, consumer worker는 `processed_events` 테이블로 중복 처리를 방지합니다.

## 테스트

```bash
make test
make vet
make test-race
```

통합 테스트:

```bash
make test-integration
```
