# 11. 성능/메모리/GC 감각 잡기

## escape analysis

Go 성능 최적화의 출발점은 "값이 스택에 머무는지, 힙으로 escape되는지" 이해하는 것입니다.

### 기본 개념

1. 스택 할당: 함수 호출 범위 내에서 빠르게 생성/해제
2. 힙 할당: GC 대상이 되어 추적/회수 비용 발생

값이 함수 밖에서도 살아야 하거나, 컴파일러가 생존 범위를 확신할 수 없으면 힙으로 escape됩니다.

예시:

```go
func newUser(name string) *User {
	u := User{Name: name}
	return &u // u가 escape되어 힙 할당 가능
}
```

### escape 확인 방법

```bash
go test -gcflags='-m' ./...
```

출력의 `escapes to heap` 메시지를 보고, 핫패스에서 불필요한 escape가 있는지 점검합니다.

주의:

1. escape 자체가 항상 나쁜 것은 아님
2. 가독성을 해치면서까지 micro-optimization 금지
3. 핫패스 + 높은 호출 빈도에서 우선 대응

## 할당 줄이기

GC 부담은 결국 "할당량"과 "객체 생명주기"에서 결정됩니다.  
고성능 경로에서는 불필요한 할당을 줄이는 것이 가장 큰 효과를 냅니다.

### 슬라이스 재사용

```go
buf := make([]byte, 0, 4096)
buf = buf[:0] // 길이만 초기화해 재사용
```

반복 처리 루프에서 매번 새 슬라이스를 만들기보다 재사용하면 할당/GC 압력이 줄어듭니다.

### 문자열 조립 최적화

반복 `+` 대신 `strings.Builder` 사용:

```go
var b strings.Builder
b.Grow(256)
b.WriteString("order=")
b.WriteString(orderID)
```

### 객체 풀(`sync.Pool`)

짧은 생명주기의 임시 객체 재사용에 유용합니다.

```go
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}
```

주의:

1. 풀은 캐시일 뿐, 영구 보관소가 아님
2. 큰 객체를 무분별하게 넣으면 메모리 피크가 커질 수 있음
3. 먼저 벤치마크로 효과 확인 후 도입

### 데이터 구조 선택

1. 작은 고정 크기 데이터는 배열/값 타입 검토
2. 맵/슬라이스는 예상 크기 사전 할당
3. 인터페이스 박싱(`any`) 남용 시 할당 증가 가능성 점검

## 프로파일링

튜닝은 반드시 "측정 -> 가설 -> 수정 -> 재측정" 순서로 진행합니다.

### CPU 프로파일

```bash
go test -run=^$ -bench=. -cpuprofile=cpu.out ./...
go tool pprof cpu.out
```

핫 함수(top)와 누적 호출 경로를 먼저 확인합니다.

### Heap 프로파일

```bash
go test -run=^$ -bench=. -memprofile=mem.out ./...
go tool pprof mem.out
```

어떤 타입/함수에서 할당이 집중되는지 파악합니다.

### Goroutine/Block/Mutex

동시성 병목은 CPU가 아니라 대기(wait)에서 발생하는 경우가 많습니다.

```bash
go test -run=^$ -bench=. -blockprofile=block.out ./...
go test -run=^$ -bench=. -mutexprofile=mutex.out ./...
```

확인 포인트:

1. lock 경쟁이 심한가
2. channel send/recv에서 block이 많은가
3. goroutine 수가 비정상적으로 증가하는가

### 운영 환경 프로파일링

서비스 운영 중에는 `net/http/pprof` 엔드포인트를 제한적으로 열어 진단할 수 있습니다.

원칙:

1. 내부망/인증 보호
2. 운영 부하에 영향 적은 시간대 진단
3. 덤프 수집 후 오프라인 분석

## 튜닝 체크리스트

성능 문제를 다룰 때는 아래 순서대로 점검합니다.

### 1) 목표 정의

1. p95/p99 지연 시간
2. 처리량(RPS)
3. 에러율/타임아웃율

목표 수치 없이 시작하면 최적화가 끝나지 않습니다.

### 2) 병목 위치 확인

1. CPU 바운드인지
2. I/O 바운드(DB/Redis/MQ/네트워크)인지
3. lock contention/GC pressure인지

### 3) 개선 액션

1. 알고리즘/쿼리 구조 개선
2. 불필요 할당 제거
3. 캐시/배치/비동기화 적용
4. 고루틴/워커 수 튜닝
5. DB 인덱스/쿼리 플랜 점검

### 4) 검증

1. 벤치마크 비교(`-benchmem`)
2. 프로파일 재수집
3. 회귀 테스트(`go test ./...`, `-race`)

### 5) 운영 반영

1. 카나리/점진 배포
2. 핵심 지표 모니터링
3. 롤백 기준 사전 정의

튜닝에서 가장 흔한 실패는 "측정 없는 변경"과 "한 번에 너무 많은 변경"입니다.

## 요약

1. escape analysis는 힙/스택 동작을 이해하는 핵심 도구다.
2. 성능 개선의 1순위는 불필요한 할당 감소다.
3. 프로파일링은 CPU뿐 아니라 Heap/Goroutine/Block/Mutex를 함께 봐야 한다.
4. 최적화는 목표 수치와 재측정 루프가 있어야 의미가 있다.

## 체크리스트

- `go test -gcflags='-m'`로 escape 지점을 확인했는가
- 핫패스에서 슬라이스/버퍼 재사용 전략을 적용했는가
- `-benchmem`과 pprof 결과를 근거로 개선하고 있는가
- lock/channel 대기로 인한 병목을 점검했는가
- 운영 반영 시 카나리/롤백 기준을 갖고 있는가

## 다음 챕터

- [12. 코드 구조화 & 아키텍처(자바 스프링 감각으로)](./12-code-structure-and-architecture.md)
