# 15. 부록

## Java <-> Go 개념/문법 매핑 표

자주 헷갈리는 개념을 빠르게 대조할 수 있도록 핵심만 정리합니다.

| Java | Go | 핵심 차이 |
|---|---|---|
| `class` | `struct` + method | 상속 대신 조합(embedding) 중심 |
| `interface implements` | 암시적 구현(interface satisfaction) | `implements` 키워드 없음 |
| `public/private/protected` | 대문자 export / 소문자 비공개 | 접근 제어가 식별자 이름 규칙으로 결정 |
| `exception (try-catch)` | `error` 반환 + 분기 | 실패 흐름이 시그니처에 명시됨 |
| `RuntimeException` | `panic` | 일반 에러 처리 용도가 아님 |
| `finally` | `defer` | 함수 종료 시점에 실행 |
| `Thread` | `goroutine` | 경량 실행 단위, 런타임 스케줄링 |
| `synchronized` | `sync.Mutex`, `RWMutex` | 명시적 락 사용 |
| `ExecutorService` | 워커 풀(worker pool) + channel + context | 패턴 조합으로 구성 |
| `Future/CompletableFuture` | channel, goroutine, errgroup 패턴 | 표준 비동기 추상화가 더 단순 |
| `Map/List` | `map` / `slice` | slice는 길이/용량 개념이 중요 |
| `null` | `nil` + zero value | zero value 설계가 중요 |
| `StringBuilder` | `strings.Builder` | 반복 문자열 조립 시 사용 |
| `Generics` | Generics(`[]T`, constraint) | 도입 시기와 범위가 더 보수적 |
| `Spring DI Container` | 생성자 주입 + main wiring | 런타임 매직보다 명시적 조립 |
| `application.yml` | env + `config struct` | 부팅 시 검증(fail fast) 권장 |
| `Maven/Gradle` | `go mod` | 표준 툴체인 단순화 |
| JUnit/Mockito | `testing` + table-driven + fake/stub | 표준 테스트 중심 |
| AOP/Filter | middleware chain | 횡단 관심사 명시 구성 |
| JVM 튜닝 | 할당/GC/pprof 튜닝 | 측정 기반 개선 필수 |

## 빠른 코드 매핑 예시

Java:

```java
public interface OrderRepository {
    Order findById(Long id);
}
```

Go:

```go
type OrderRepository interface {
	FindByID(ctx context.Context, id int64) (Order, error)
}
```

Java:

```java
try {
    service.createOrder(req);
} catch (NotFoundException e) {
    ...
}
```

Go:

```go
if err := service.CreateOrder(ctx, req); err != nil {
	if errors.Is(err, domain.ErrNotFound) {
		...
	}
}
```

## 자주 하는 실수 TOP 20

Go 입문자가 실무에서 자주 부딪히는 문제를 우선순위 순서로 정리했습니다.

1. `nil map`에 쓰기 시도(`panic`)
2. 슬라이스 복사 없이 공유해서 원본이 예상치 않게 변경됨
3. `len(string)`을 문자 수로 오해(실제는 바이트 수)
4. `error`를 문자열 비교로 분기
5. `%w` 없이 래핑해서 원인 추적(`errors.Is/As`)이 깨짐
6. 일반 에러 처리에 `panic` 남용
7. `context`를 I/O 경계(DB/HTTP/Redis/MQ)에 전달하지 않음
8. `context.WithTimeout` 생성 후 `cancel()` 누락
9. goroutine 종료 조건이 없어 누수 발생
10. 채널 close 책임이 불명확해서 데드락 발생
11. map 동시 쓰기를 락 없이 수행
12. `WaitGroup.Add`/`Done` 호출 순서 실수
13. receiver(value/pointer)를 타입 내에서 혼용
14. 인터페이스를 구현 전 과도하게 크게 설계
15. `nil interface` 함정(`var err error = (*MyErr)(nil)`)
16. 트랜잭션에서 외부 I/O를 수행해 락 유지 시간이 길어짐
17. 준비되지 않은 JSON 디코딩(unknown field(알 수 없는 필드) 허용으로 계약 오염)
18. 로그에 `request_id`/`trace_id`가 없어 추적 불가
19. 성능 문제를 측정 없이 체감으로 최적화
20. 배포 전/후 체크리스트 없이 릴리즈

## 실수 예방 공통 규칙

1. `go test ./...` + `go test -race ./...`를 기본 루틴으로 실행
2. API 경계에서 입력 검증/에러 매핑/로깅 표준화
3. 코드 리뷰에서 "의존 방향, 컨텍스트(context) 전달, 종료 조건"을 필수 점검

## 추천 레퍼런스/스타일 가이드/읽을거리

아래 자료는 실무 신뢰도가 높고, 장기적으로 다시 참고하기 좋은 자료들입니다.

### 공식 문서

1. Go 공식 문서: [https://go.dev/doc/](https://go.dev/doc/)
2. Effective Go: [https://go.dev/doc/effective_go](https://go.dev/doc/effective_go)
3. Go by Example: [https://gobyexample.com/](https://gobyexample.com/)
4. 표준 라이브러리 문서: [https://pkg.go.dev/std](https://pkg.go.dev/std)

### 언어/설계 심화

1. Go Blog: [https://go.dev/blog/](https://go.dev/blog/)
2. Memory Model: [https://go.dev/ref/mem](https://go.dev/ref/mem)
3. Modules Reference: [https://go.dev/ref/mod](https://go.dev/ref/mod)

### 스타일/리뷰 가이드

1. Google Go Style Decisions: [https://google.github.io/styleguide/go/decisions](https://google.github.io/styleguide/go/decisions)
2. Uber Go Style Guide: [https://github.com/uber-go/guide](https://github.com/uber-go/guide)
3. Code Review Comments: [https://go.dev/wiki/CodeReviewComments](https://go.dev/wiki/CodeReviewComments)

### 운영/관측성

1. OpenTelemetry: [https://opentelemetry.io/docs/](https://opentelemetry.io/docs/)
2. Prometheus 문서: [https://prometheus.io/docs/introduction/overview/](https://prometheus.io/docs/introduction/overview/)

## 추천 학습 루프

1. 문서 읽기(개념)
2. 예제 실행(작동)
3. 프로젝트 반영(적용)
4. 테스트/벤치/프로파일(검증)
5. 회고 후 규칙 문서화(팀 표준화)

## 예제 프로젝트 전체 소스 구조

책에서 점진적으로 완성하는 `go-commerce-api`의 권장 최종 구조 예시는 아래와 같습니다.

```text
go-commerce-api/
├─ cmd/
│  └─ api/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ domain/
│  │  ├─ order.go
│  │  └─ errors.go
│  ├─ service/
│  │  └─ order_service.go
│  ├─ repository/
│  │  ├─ postgres/
│  │  │  └─ store.go
│  │  └─ redis/
│  │     └─ cache.go
│  ├─ transport/
│  │  └─ http/
│  │     ├─ handler_order.go
│  │     ├─ middleware_auth.go
│  │     └─ middleware_recover.go
│  ├─ messaging/
│  │  └─ nats/
│  │     └─ publisher.go
│  └─ worker/
│     ├─ outbox_publisher.go
│     └─ order_event_worker.go
├─ migrations/
│  ├─ 001_init.sql
│  └─ 002_order_index.sql
├─ deployments/
│  ├─ Dockerfile
│  ├─ docker-compose.yml
│  └─ k8s/
│     ├─ deployment.yaml
│     └─ service.yaml
├─ test/
│  ├─ integration/
│  │  └─ order_flow_test.go
│  └─ fixtures/
├─ docs/
│  └─ api/
│     └─ openapi.yaml
├─ go.mod
├─ go.sum
└─ Makefile
```

## 운영 파일 권장 추가 항목

1. `Makefile`: `test`, `lint`, `bench`, `run` 명령 표준화
2. `.golangci.yml`: 린트 규칙 고정
3. `README.md`: 실행/테스트/배포 절차
4. `CHANGELOG.md`: 릴리즈 변경 이력
5. `docs/runbook.md`: 장애 대응 절차

## 요약

1. Java와 Go는 문법보다 실패 처리, 동시성, 의존성 관리 방식에서 큰 차이가 있습니다.
2. 실무 품질은 언어 지식 자체보다 테스트/관측성/배포 규율에서 결정됩니다.
3. 마이그레이션과 운영은 "점진 전환 + 계약 호환 + 즉시 롤백" 원칙이 핵심입니다.
4. 예제 프로젝트 구조를 기준으로 팀 표준을 문서화하면 유지보수 비용을 크게 줄일 수 있습니다.

## 체크리스트

- 00~15 챕터가 목차 순서대로 존재하는가
- 챕터 성격에 맞게 `요약`/`체크리스트` 구성이 일관되게 적용되어 있는가
- 예제 코드 경로와 문서 설명이 일관되는가
- 운영 관점(테스트/성능/배포/관측성)이 누락되지 않았는가

## 다음 챕터

- 마지막 챕터입니다. [00. 이 가이드의 목표와 전제](./00-goals-and-assumptions.md)로 돌아가기
