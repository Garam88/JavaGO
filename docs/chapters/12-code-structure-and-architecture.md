# 12. 코드 구조화 & 아키텍처(자바 스프링 감각으로)

## 패키지 설계 원칙

Go에서 아키텍처의 핵심은 "패키지 경계로 의존 방향을 통제"하는 것입니다.  
Java/Spring에서 계층을 클래스/어노테이션 중심으로 나누던 방식과 달리, Go는 폴더와 import 관계가 설계 그 자체입니다.

`00`장에서 정의한 `go-commerce-api` 구조를 기준으로 보면:

```text
internal/
├─ domain/
├─ service/
├─ repository/
├─ transport/http/
└─ worker/
```

### 도메인 중심 vs 레이어 중심

1. 레이어 중심(Controller/Service/Repository 분리)
- 장점: 익숙하고 시작이 빠름
- 단점: 기능이 늘면 레이어 간 파일 점프가 많아짐

2. 도메인 중심(order/user/payment 등)
- 장점: 기능 응집도 높음, 팀 단위 분할에 유리
- 단점: 공통 인프라 추출 기준이 필요

실무 권장:

1. 초기에는 레이어 중심으로 단순 시작
2. 기능 커지면 도메인 경계 단위로 재편
3. 어떤 방식이든 "의존 방향"을 고정

### 의존 방향 규칙

1. `transport` -> `service` -> `repository` (한 방향)
2. `domain`은 인프라 세부사항을 모름
3. 순환 import는 설계 경고 신호로 간주

이 규칙만 지켜도 코드베이스가 커질 때 구조 붕괴를 크게 줄일 수 있습니다.

## DI 접근

Go에는 Spring처럼 런타임 DI 컨테이너가 표준으로 존재하지 않습니다.  
대신 명시적인 생성자 주입이 기본입니다.

### 생성자 주입 기본

```go
type OrderService struct {
	repo OrderRepository
	bus  EventBus
}

func NewOrderService(repo OrderRepository, bus EventBus) *OrderService {
	return &OrderService{repo: repo, bus: bus}
}
```

장점:

1. 의존성이 시그니처에 드러남
2. 테스트 대역 주입이 쉬움
3. 런타임 매직이 없어 디버깅이 단순

### Composition Root

의존성 조립은 보통 `cmd/api/main.go`에서 한 번에 수행합니다.

1. config 로드
2. infra(DB/Redis/NATS) 생성
3. repository/service/handler 연결
4. server 시작

이 파일이 "애플리케이션 wiring의 단일 진입점" 역할을 합니다.

### `wire` 같은 도구 사용 시점

다음 조건에서 코드 생성 기반 DI를 고려할 수 있습니다.

1. 서비스 수/의존성이 커져 수동 wiring이 과도할 때
2. 팀이 코드 생성 워크플로우를 안정적으로 운영할 수 있을 때
3. 에러 메시지/빌드 흐름 복잡도를 수용할 수 있을 때

초기 단계에서는 수동 생성자 주입이 가장 단순하고 유지보수 비용이 낮습니다.

## 설정 관리

Go 서비스 설정은 "환경 변수 -> config struct -> 검증" 흐름이 가장 일반적입니다.

### config struct 패턴

```go
type Config struct {
	HTTPPort           string
	DBDSN              string
	RedisAddr          string
	NATSURL            string
	RequestTimeout     time.Duration
	CacheTTL           time.Duration
	OutboxPollInterval time.Duration
}
```

```go
func LoadConfig() (Config, error) {
	cfg := Config{
		HTTPPort:  getenv("HTTP_PORT", "8080"),
		DBDSN:     os.Getenv("DB_DSN"),
		RedisAddr: getenv("REDIS_ADDR", "localhost:6379"),
		NATSURL:   getenv("NATS_URL", "nats://localhost:4222"),
	}
	if cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required")
	}
	return cfg, nil
}
```

### 설정 원칙

1. 필수값은 부팅 시 실패(fail fast)
2. 기본값은 안전한 범위에서만 제공
3. 타입 변환/검증 로직을 중앙화
4. 비밀값은 로그에 출력 금지

### 환경별 분리

1. dev/stage/prod를 환경 변수 세트로 분리
2. 코드 분기보다 설정 분기로 동작 차이 제어
3. 배포 파이프라인에서 설정 주입 이력 관리

설정이 흩어지면 운영 장애 시 원인 추적이 어려워집니다.  
`internal/config` 같은 단일 패키지로 모아두는 것이 안전합니다.

## 인터페이스 경계 설계

Spring의 Port/Adapter(헥사고날) 개념은 Go에서도 유효하지만, "작고 필요한 인터페이스"로 좁혀 쓰는 것이 핵심입니다.

### 포트 정의 위치

인터페이스는 보통 "사용하는 쪽"에 둡니다.

```go
// service 패키지
type OrderRepo interface {
	Save(ctx context.Context, o Order) error
	FindByID(ctx context.Context, id int64) (Order, error)
}
```

이렇게 하면 구현 세부사항보다 유스케이스 요구사항 중심으로 경계를 설계할 수 있습니다.

### 어댑터 구현

1. inbound adapter: HTTP handler, NATS consumer
2. outbound adapter: Postgres repo, Redis client, message publisher

도메인/서비스는 어댑터 구현체를 몰라도 동작해야 합니다.

### 안티패턴

1. 구현 전 광범위 인터페이스부터 정의
2. 모든 계층에 인터페이스를 습관적으로 추가
3. 에러 타입/트랜잭션 정책을 경계마다 제각각 처리

### 실무 적용 순서

1. 유스케이스를 먼저 코드로 작성
2. 테스트를 위해 필요한 최소 인터페이스 추출
3. adapter 구현(Postgres/Redis/NATS/HTTP) 연결
4. 경계별 에러/타임아웃/로깅 규칙 통일

`go-commerce-api`에 적용하면:

1. 주문 생성 유스케이스는 `Store`와 `Cache` 포트만 의존
2. HTTP handler는 요청/응답 변환만 담당
3. Postgres/Redis/NATS 어댑터는 composition root(`cmd/api/main.go`)에서 조립
4. 인프라 교체(Postgres -> 다른 저장소) 영향이 서비스 코어로 전파되지 않음

## 요약

1. Go 아키텍처의 핵심은 패키지 경계와 의존 방향 통제다.
2. DI는 생성자 주입 + main wiring으로 충분히 견고하게 운영할 수 있다.
3. 설정은 환경 변수 기반 `config struct`로 중앙화하고 부팅 시 검증해야 한다.
4. 헥사고날은 작은 인터페이스와 명확한 어댑터 경계로 적용할 때 효과적이다.

## 체크리스트

- 패키지 import 방향이 단방향으로 유지되는가
- 의존성 생성이 `cmd/.../main.go`에서 일관되게 조립되는가
- 필수 설정 누락 시 애플리케이션이 즉시 실패하는가
- 인터페이스가 유스케이스 최소 단위로 유지되는가
- 인프라 변경 영향이 도메인 코어로 새지 않도록 경계가 설계되어 있는가

## 다음 챕터

- [13. 배포/운영: 실무 필수](./13-deployment-and-operations.md)
