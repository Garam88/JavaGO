# 09. 데이터 계층: DB/캐시/메시징

## database/sql과 커넥션 풀

Go 표준의 `database/sql`은 드라이버 추상화 + 커넥션 풀을 제공합니다.  
핵심은 "풀을 직접 관리하는 것이 아니라 설정을 통해 제어"하는 것입니다.

### 기본 연결

```go
db, err := sql.Open("postgres", dsn)
if err != nil {
	return err
}
if err := db.PingContext(ctx); err != nil {
	return err
}
```

`sql.Open`은 실제 연결 검증이 아니라 설정 생성에 가깝습니다.  
초기 부팅 검증은 `PingContext`로 확인해야 합니다.

### 풀 설정

```go
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(25)
db.SetConnMaxIdleTime(5 * time.Minute)
db.SetConnMaxLifetime(30 * time.Minute)
```

운영 체크:

1. `MaxOpenConns`가 DB 서버 한계를 넘지 않게 설정
2. 너무 작은 값은 대기 증가, 너무 큰 값은 DB 과부하
3. 수명(`MaxLifetime`)을 두어 오래된 연결을 주기적으로 교체

### 컨텍스트 전달

모든 쿼리는 `QueryContext`/`ExecContext`로 timeout/cancel을 전파합니다.

```go
rows, err := db.QueryContext(ctx, q, args...)
```

`context` 누락은 장애 시 지연 확산으로 이어집니다.

## 트랜잭션/Prepared Statement/스캐닝

트랜잭션은 "업무 단위의 원자성"을 보장하는 경계입니다.  
`go-commerce-api`에서는 주문 저장 + 재고 차감 같은 묶음 처리에 필수입니다.

### 트랜잭션 패턴

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
	return err
}
defer tx.Rollback()

if _, err := tx.ExecContext(ctx, insertOrderQ, ...); err != nil {
	return err
}
if _, err := tx.ExecContext(ctx, updateStockQ, ...); err != nil {
	return err
}
if err := tx.Commit(); err != nil {
	return err
}
```

원칙:

1. `BeginTx` -> `defer Rollback` -> 성공 시 `Commit`
2. 트랜잭션 내부에서 외부 네트워크 호출 최소화
3. 락 유지 시간을 줄이기 위해 쿼리 순서와 범위를 관리

### Prepared Statement

반복 실행 쿼리는 `PrepareContext`로 성능/안전성을 확보할 수 있습니다.

```go
stmt, err := db.PrepareContext(ctx, `SELECT id, status FROM orders WHERE id = $1`)
if err != nil {
	return err
}
defer stmt.Close()
```

실무에서는 드라이버/DB 특성상 성능 차이가 크지 않을 수도 있으므로, 벤치마크와 모니터링으로 판단하는 것이 안전합니다.

### 스캐닝 패턴

```go
var o Order
err := row.Scan(&o.ID, &o.Status, &o.CreatedAt)
```

주의:

1. 컬럼 순서와 Scan 대상 순서를 반드시 맞춤
2. `NULL` 가능 컬럼은 `sql.NullString`/`sql.NullTime` 또는 포인터로 처리
3. `sql.ErrNoRows`는 도메인 `ErrNotFound`로 매핑

## ORM/쿼리빌더 선택 기준

Go에서 데이터 접근은 크게 세 가지 선택지가 있습니다.

1. 순수 SQL (`database/sql`)
2. 쿼리 빌더 (`squirrel`, `goqu` 등)
3. ORM (`gorm`, `ent` 등)

### 선택 기준

1. 쿼리 복잡도
2. 팀 SQL 숙련도
3. 성능/튜닝 요구 수준
4. 마이그레이션/스키마 관리 전략

### 트레이드오프

1. 순수 SQL
- 장점: 제어력 높음, SQL이 명확
- 단점: 보일러플레이트 증가

2. 쿼리 빌더
- 장점: 동적 쿼리 구성 편함
- 단점: 복잡해지면 SQL 가독성 저하

3. ORM
- 장점: 생산성, CRUD 속도
- 단점: 숨은 쿼리/성능 디버깅 난이도

`go-commerce-api`처럼 성능·운영 가시성이 중요한 서비스는 "핵심 경로는 SQL 명시, 반복 CRUD는 선택적 추상화" 전략이 실무에서 안정적입니다.

## Redis/캐시 패턴

Redis는 읽기 부하 완화와 응답 지연 감소에 효과적이지만, 일관성 정책 없이 도입하면 오히려 장애 원인이 됩니다.

### 기본 캐시 패턴

1. Cache-Aside
- 조회: 캐시 miss면 DB 조회 후 캐시에 저장
- 갱신: DB 쓰기 후 캐시 무효화 또는 갱신

2. Write-Through/Write-Behind
- 구현 복잡도와 장애 전파 위험이 커서 초기에 신중히 선택

### 캐시 키 설계

```text
order:{order_id}
user:{user_id}:orders:page:{n}
```

규칙:

1. 도메인 prefix 일관성 유지
2. TTL 정책 명시
3. 버전 키(`v1:`)로 스키마 변경 대비

### 캐시 스탬피드 대응

1. 적절한 TTL + 랜덤 지터
2. 요청 합치기(singleflight)
3. hot key 보호 전략

### 분산 락과 멱등성

중복 처리 방지(결제/주문 생성)에는 Redis 락 또는 idempotency key를 사용합니다.

1. 락은 TTL 필수
2. 락 해제는 소유자 검증 필요
3. 비즈니스 멱등성은 DB unique 제약과 함께 사용

실무에서는 "Redis 락만 믿지 않고 DB 제약으로 최종 보장"이 안전합니다.

## 메시지 큐/이벤트 처리

메시지 큐는 비동기 처리와 서비스 분리를 돕지만, 전달 보장과 중복 처리 전략을 반드시 설계해야 합니다.

### 컨슈머 워커 기본 구조

1. 메시지 수신
2. 핸들러 처리
3. 성공 ack / 실패 retry
4. 임계 실패 시 DLQ

`07`장에서 다룬 워커 풀(worker pool) + `context` 취소 구조를 그대로 적용합니다.

### 재시도 전략

1. 즉시 재시도 남용 금지
2. 지수 백오프(exponential backoff)
3. 최대 시도 횟수 초과 시 DLQ 이동

재시도 가능한 오류(일시 장애)와 불가능한 오류(검증 실패)를 분리해야 무한 루프를 막을 수 있습니다.

### 멱등성(idempotency)

메시지는 중복 전달될 수 있다는 전제로 설계합니다.

1. 이벤트 ID 저장(처리 이력 테이블/키)
2. 이미 처리한 ID면 noop
3. 상태 전이 조건을 명시적으로 검증

### 트랜잭션과 이벤트 발행

DB 업데이트와 이벤트 발행의 원자성을 맞추려면 Outbox 패턴을 고려합니다. 현재 `go-commerce-api` 예제는 주문 생성 트랜잭션 안에서 재고 차감, 주문 저장, `outbox_events` insert를 함께 수행하고, 별도 publisher가 pending 이벤트를 NATS JetStream으로 발행합니다.

1. 비즈니스 변경 + outbox insert를 같은 트랜잭션으로 처리
2. 별도 퍼블리셔가 outbox를 읽어 브로커에 발행
3. 발행 성공 후 outbox 상태 갱신

이 패턴은 "DB 성공, 메시지 실패" 불일치 문제를 크게 줄여줍니다.

예제의 기준 구현:

1. DB: PostgreSQL + `database/sql` + pgx stdlib
2. Cache: Redis cache-aside, 키 prefix는 `v1:item:{id}`, `v1:order:{id}`
3. Message: NATS JetStream, stream은 `ORDER_EVENTS`, subject는 `order.created`
4. Consumer: `processed_events` 테이블에 이벤트 ID를 기록해 중복 처리를 방지

## 요약

1. `database/sql`은 풀 설정과 컨텍스트 전파가 품질의 핵심이다.
2. 트랜잭션 경계를 명확히 하고 `ErrNoRows` 같은 DB 에러를 도메인 에러로 매핑해야 한다.
3. ORM/쿼리빌더 선택은 생산성과 제어력의 균형 문제다.
4. Redis는 캐시 정책, TTL, 멱등성, 락 안전장치를 함께 설계해야 한다.
5. 메시지 처리는 재시도/DLQ/Outbox/멱등성까지 포함해 운영 관점으로 설계해야 한다.

## 체크리스트

- DB 커넥션 풀 파라미터를 근거 있게 설정했는가
- 모든 DB 호출에 `Context` 타임아웃이 전달되는가
- 트랜잭션 범위가 불필요하게 길지 않은가
- 캐시 키/TTL/무효화 정책이 문서화되어 있는가
- 메시지 소비 로직이 중복 처리와 재시도 한계를 고려하는가

## 다음 챕터

- [10. 테스트 전략: Java(JUnit/Mockito) 관점에서](./10-testing-strategy-from-junit.md)
