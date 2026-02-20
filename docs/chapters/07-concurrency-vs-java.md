# 07. 동시성(Go의 킬러 기능): Java 스레드/Executor와 비교

## goroutine vs thread

Java 개발자가 Go 동시성에서 가장 먼저 체감하는 차이는 "생성 비용과 운영 방식"입니다.

1. Java thread: OS 스레드와 가까운 단위
2. Go goroutine: 런타임 스케줄러가 관리하는 경량 실행 단위

Go 런타임은 많은 goroutine을 소수의 OS 스레드에 매핑해 실행합니다.  
그래서 요청 단위 작업을 goroutine으로 쪼개도 상대적으로 부담이 적습니다.

```go
go func() {
	processOrder(ctx, orderID)
}()
```

주의할 점:

1. "가볍다"는 "무제한 생성 가능"을 의미하지 않습니다.
2. 고루틴 누수는 메모리/핸들/CPU 문제로 바로 이어집니다.
3. 종료 조건(`context`, channel close, done signal)을 반드시 설계해야 합니다.

실무에서는 아래 원칙이 안전합니다.

1. 수명 짧은 작업은 요청 스코프에서 생성
2. 장수 워커는 명시적 시작/종료 루프 운영
3. shutdown 시 모든 goroutine 종료를 보장

## 채널 기본과 패턴

채널(channel)은 goroutine 간 데이터 전달과 동기화를 함께 해결하는 도구입니다.

### 기본

```go
jobs := make(chan Job)
results := make(chan Result)
```

`chan T`는 타입 안정성을 유지하면서 메시지를 전달합니다.

### 버퍼드 vs 언버퍼드

1. 언버퍼드: 송신/수신이 만나야 진행(강한 동기화)
2. 버퍼드: 버퍼 크기만큼 비동기 완충

```go
queue := make(chan Job, 100)
```

버퍼를 크게 잡는 것이 항상 좋은 것은 아닙니다.  
병목이 숨겨져 지연이 늦게 드러날 수 있습니다.

### 팬아웃/팬인

1. 팬아웃: 하나의 입력을 여러 worker가 분산 처리
2. 팬인: 여러 worker 결과를 하나로 수집

```go
for i := 0; i < workerN; i++ {
	go worker(ctx, jobs, results)
}
```

### 워커풀 패턴

`go-commerce-api`의 이벤트 컨슈머(예: `order.created`)에 자주 적용합니다.

1. 입력 큐 채널
2. 고정 수 worker goroutine
3. 종료 신호(context cancel)
4. 재시도/실패 채널 또는 DLQ 분기

핵심 체크:

1. producer가 끝났을 때 `jobs`를 닫는가
2. consumer는 channel close를 감지하고 종료하는가
3. 결과 수집이 block되어 goroutine이 멈추지 않는가

## Context 기반 취소/타임아웃/전파

`context.Context`는 Go에서 취소, 데드라인, 요청 스코프 데이터를 전파하는 표준입니다.

Java의 interrupt/timeout 관리와 비슷한 목적이지만, Go에서는 함수 시그니처에 명시적으로 전달합니다.

```go
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) error
```

### 기본 패턴

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()

if err := repo.Save(ctx, order); err != nil {
	return err
}
```

원칙:

1. I/O 경계(DB, HTTP, Redis, MQ)에는 항상 `ctx` 전달
2. `WithCancel`/`WithTimeout`를 만들면 `cancel()`을 반드시 호출
3. `context.Background()`는 진입점(main/test)에서만 주로 사용

### 취소 처리 패턴

```go
select {
case <-ctx.Done():
	return ctx.Err()
case msg := <-ch:
	return handle(msg)
}
```

### Context Value 주의

`context.WithValue`는 최소한으로 사용합니다.

1. `request_id`, `trace_id` 같은 횡단 관심사
2. 비즈니스 필수 데이터 전달 용도로 남용 금지

필수 의존성은 함수 인자로 명시하는 편이 테스트와 가독성에 유리합니다.

## 동기화 도구

공유 메모리를 다루는 경우 채널만으로 충분하지 않을 때가 많습니다.  
그때는 `sync` 패키지 도구를 명확한 목적에 맞게 씁니다.

### `sync.Mutex`

공유 데이터에 대한 배타적 접근이 필요할 때 사용합니다.

```go
mu.Lock()
state.count++
mu.Unlock()
```

`defer mu.Unlock()`은 안전하지만 핫패스에서는 비용을 검토하세요.

### `sync.RWMutex`

읽기 많고 쓰기 적은 구조에서 유리합니다.

```go
mu.RLock()
v := cache[key]
mu.RUnlock()
```

### `sync.WaitGroup`

여러 goroutine 완료를 기다릴 때 사용합니다.

```go
var wg sync.WaitGroup
wg.Add(n)
for i := 0; i < n; i++ {
	go func() {
		defer wg.Done()
		work()
	}()
}
wg.Wait()
```

실수 방지:

1. `Add`는 goroutine 시작 전에 호출
2. `Done` 누락 금지

### `sync.Once`

지연 초기화(예: 전역 클라이언트 생성)에 사용합니다.

```go
once.Do(func() {
	client = newClient()
})
```

### 도구 선택 기준

1. 메시지 전달/파이프라인 중심: channel
2. 공유 상태 보호: Mutex/RWMutex
3. 완료 동기화: WaitGroup
4. 단 한 번 초기화: Once

## 디버깅

동시성 버그는 재현이 어렵기 때문에 "도구 기반 검증"이 필수입니다.

### 레이스 디텍터

```bash
go test -race ./...
```

CI에도 반드시 포함하는 것을 권장합니다.

### 데드락/블로킹 점검

1. goroutine dump로 대기 지점 확인
2. channel send/recv 짝이 맞는지 확인
3. lock 획득 순서 일관성 점검

운영 중에는 블록/뮤텍스 프로파일을 활용합니다.

```bash
go test -run=^$ -bench=. -blockprofile=block.out ./...
go test -run=^$ -bench=. -mutexprofile=mutex.out ./...
```

### 흔한 장애 패턴

1. 닫힌 채널에 send
2. 닫히지 않는 채널 range
3. context cancel 누락으로 goroutine 누수
4. lock 순서 역전으로 교착상태
5. shared map 락 없는 write

### 점검 루틴(권장)

1. PR 단계: `go test -race ./...`
2. 성능 이슈: 벤치마크 + block/mutex profile
3. 배포 전: graceful shutdown 시 goroutine 종료 확인

## 요약

1. goroutine은 경량이지만 종료 설계 없이는 쉽게 누수된다.
2. channel은 메시지 전달과 동기화를 단순화하지만 close/버퍼 전략이 중요하다.
3. `context`는 취소/타임아웃 전파의 표준이며 I/O 경계에서 필수다.
4. 공유 상태는 `sync` 도구로 명확히 보호해야 한다.
5. 동시성 품질은 `-race`와 프로파일링 루틴으로 검증해야 한다.

## 체크리스트

- goroutine 생성 지점마다 종료 조건을 갖고 있는가
- 채널 close 책임이 코드에서 명확한가
- DB/HTTP/Redis 호출에 `ctx`가 전달되는가
- 맵/카운터 같은 공유 상태를 락으로 보호하고 있는가
- `go test -race ./...`를 정기 실행하고 있는가

## 다음 챕터

- [08. 표준 라이브러리로 만드는 서버 개발(웹 백엔드)](./08-server-development-with-stdlib.md)
