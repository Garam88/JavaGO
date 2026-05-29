# Go 입문서 프로젝트

- 이 저장소는 Markdown 기반 Go 입문서 원고와 실행 가능한 예제 코드를 함께 관리합니다.
- 이 프로젝트는 GPT-5.3-Codex를 활용하여 제작되었습니다. 

## 디렉터리 구조

- `docs/`: 책 원고 작업 공간 (`docs/chapters` 중심)
- `examples/`: 실행 가능한 예제 프로젝트 (`go-commerce-api`)
- `Makefile`: 자주 쓰는 실행 명령 모음

## 전체 챕터 목차

- [00. 이 가이드의 목표와 전제](docs/chapters/00-goals-and-assumptions.md)
- [01. Go 한눈에 보기: Java 개발자가 먼저 알아야 할 핵심](docs/chapters/01-go-at-a-glance-for-java.md)
- [02. 개발 환경 세팅 & 툴체인](docs/chapters/02-setup-and-toolchain.md)
- [03. 언어 기초 문법(자바 대비 포인트 중심)](docs/chapters/03-language-basics-for-java.md)
- [04. 타입 시스템: Java 개발자가 헷갈리는 지점 정리](docs/chapters/04-type-system-pitfalls-for-java.md)
- [05. 컬렉션/문자열/유틸 실전](docs/chapters/05-collections-strings-utils.md)
- [06. 예외가 없는 세계: 에러 처리/리커버리 패턴](docs/chapters/06-error-handling-and-recovery.md)
- [07. 동시성(Go의 킬러 기능): Java 스레드/Executor와 비교](docs/chapters/07-concurrency-vs-java.md)
- [08. 표준 라이브러리로 만드는 서버 개발(웹 백엔드)](docs/chapters/08-server-development-with-stdlib.md)
- [09. 데이터 계층: DB/캐시/메시징](docs/chapters/09-data-layer-db-cache-messaging.md)
- [10. 테스트 전략: Java(JUnit/Mockito) 관점에서](docs/chapters/10-testing-strategy-from-junit.md)
- [11. 성능/메모리/GC 감각 잡기](docs/chapters/11-performance-memory-gc.md)
- [12. 코드 구조화 & 아키텍처(자바 스프링 감각으로)](docs/chapters/12-code-structure-and-architecture.md)
- [13. 배포/운영: 실무 필수](docs/chapters/13-deployment-and-operations.md)
- [14. Java -> Go 마이그레이션 가이드(선택)](docs/chapters/14-java-to-go-migration-guide.md)
- [15. 부록](docs/chapters/15-appendix.md)

## 예시 코드 빠른 시작

예제 API는 Postgres, Redis, NATS JetStream을 사용합니다. 가장 간단한 실행 방법은 Docker Compose입니다.

```bash
docker compose -f examples/docker-compose.yml up --build
```

의존성만 먼저 띄우고 로컬에서 API를 실행할 수도 있습니다.

```bash
docker compose -f examples/docker-compose.yml up -d postgres redis nats
make run
```

기본 API:

- `GET /livez`
- `GET /readyz`
- `GET /items`
- `GET /items/{id}`
- `POST /orders`
- `GET /orders`
- `GET /orders/{id}`

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"u-1","item_id":"sku-1","quantity":2}'
```

포트 충돌 시:

```bash
API_PORT=18080 docker compose -f examples/docker-compose.yml up --build
```

## 라이선스

- 문서: `CC BY-NC-SA 4.0`
- 예제 코드: `MIT`
