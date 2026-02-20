# go-commerce-api

`examples/go-commerce-api`는 문서의 예제 프로젝트를 실행 가능한 최소 형태로 만든 스켈레톤입니다.

## 로컬 실행

```bash
cd examples/go-commerce-api
go run ./cmd/api
```

기본 포트는 `8080`이며, 환경 변수 `HTTP_PORT`로 변경할 수 있습니다.

## Docker Compose 실행

프로젝트 루트(`go-book`)에서 아래 명령으로 API + Postgres + Redis를 함께 실행할 수 있습니다.

```bash
docker compose -f examples/docker-compose.yml up --build
```

기본 공개 포트는 `8080`이고, 충돌 시 아래처럼 변경할 수 있습니다.

```bash
API_PORT=18080 docker compose -f examples/docker-compose.yml up --build
```

중지:

```bash
docker compose -f examples/docker-compose.yml down
```

데이터 볼륨까지 제거:

```bash
docker compose -f examples/docker-compose.yml down -v
```

참고:

- 현재 스켈레톤 API는 in-memory 저장소를 사용합니다.
- Compose의 Postgres/Redis는 이후 챕터 확장(DB/캐시 연동) 기준 환경으로 함께 띄우도록 구성되어 있습니다.
- Postgres/Redis 포트는 호스트에 노출하지 않고 Compose 내부 네트워크에서만 접근합니다.

## 기본 API

- `GET /healthz`
- `POST /orders`
- `GET /orders`
- `GET /orders/{id}`

### 주문 생성 예시

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"u-1","item_id":"sku-1","quantity":2}'
```
