# 06. 예외가 없는 세계: 에러 처리/리커버리 패턴

## 에러 생성/래핑/언래핑

Go에서 에러는 예외가 아니라 값입니다.  
핵심은 "실패 맥락을 잃지 않고 호출자에게 전달"하는 것입니다.

### 에러 생성

단순 상태 신호는 `errors.New`를 사용합니다.

```go
var ErrNotFound = errors.New("not found")
```

입력값 등 런타임 정보가 필요하면 `fmt.Errorf`를 사용합니다.

```go
return fmt.Errorf("invalid user_id: %d", userID)
```

### 에러 래핑(wrap)

하위 계층 에러에 상위 맥락을 추가할 때 `%w`를 사용합니다.

```go
order, err := repo.Find(ctx, id)
if err != nil {
	return nil, fmt.Errorf("find order id=%d: %w", id, err)
}
```

`%v`가 아니라 `%w`를 써야 `errors.Is`/`errors.As`로 원인 탐색이 가능합니다.

### 언래핑(unwrapping)과 분기

```go
if errors.Is(err, sql.ErrNoRows) {
	return ErrNotFound
}

var ve *ValidationError
if errors.As(err, &ve) {
	return fmt.Errorf("validation failed: %w", ve)
}
```

실무 규칙:

1. 하위 에러를 덮어쓰지 말고 래핑해 맥락을 보강
2. 분기할 때는 문자열 비교 대신 `errors.Is`/`errors.As`
3. 로그는 경계에서 1회, 반환은 래핑 중심으로

### Sentinel vs Typed Error

1. Sentinel (`var ErrX = errors.New(...)`)
장점: 단순하고 비교 쉬움
단점: 상세 정보 확장 어려움

2. Typed (`type ValidationError struct { ... }`)
장점: 필드 기반 분기/응답 생성 가능
단점: 타입이 과도하게 늘 수 있음

사용 기준:

1. 상태 신호만 필요하면 Sentinel
2. 코드/필드/원인 등 메타데이터가 필요하면 Typed
3. 외부 노출 전에는 도메인 에러로 정규화

## 자바 try-catch 대비

Java와 Go의 가장 큰 차이는 실패 처리의 "위치"입니다.

- Java: `throw` -> 상위 `try-catch`에서 제어
- Go: `error` 반환 -> 호출 지점에서 즉시 판단

Go에서는 흐름이 더 직선적이고 명시적입니다.

```go
user, err := svc.GetUser(ctx, id)
if err != nil {
	return nil, err
}
```

장점:

1. 함수 시그니처만 봐도 실패 가능성이 드러남
2. 숨은 제어 흐름이 적어 디버깅이 쉬움
3. 계층 경계에서 에러 정책(매핑/로그/메트릭) 제어가 명확

단점처럼 느껴지는 부분:

1. `if err != nil` 반복이 많아 보임
2. 초기에 코드가 장황하게 보일 수 있음

하지만 반복을 줄이는 올바른 방법은 "예외처럼 숨기기"가 아니라, 함수 분해와 공통 유틸(에러 매핑, 응답 변환)로 중복을 줄이는 것입니다.

### 계층별 에러 처리 원칙

1. Repository 계층
- DB/외부 에러를 래핑해서 상위로 전달

2. Service 계층
- 도메인 규칙 위반을 도메인 에러로 변환

3. Handler/Transport 계층
- 도메인 에러를 HTTP status/응답 바디로 매핑

예시:

```go
if errors.Is(err, domain.ErrNotFound) {
	writeJSON(w, http.StatusNotFound, ...)
	return
}
writeJSON(w, http.StatusInternalServerError, ...)
```

## panic/recover 사용 기준

`panic`은 일반 비즈니스 에러 처리 도구가 아닙니다.  
"프로세스를 중단해야 할 정도의 비정상 상태" 또는 "복구 경계"에서만 사용합니다.

### `panic`을 써도 되는 경우

1. 절대 발생하면 안 되는 프로그래밍 오류(불변식 파괴)
2. 애플리케이션 부팅 실패(필수 설정 누락 등)
3. 라이브러리 내부에서 내부 불변식 위반 탐지(문서화 필수)

### `panic`을 피해야 하는 경우

1. 사용자 입력 검증 실패
2. DB 조회 실패/타임아웃
3. 네트워크 일시 오류

위 항목은 모두 `error`로 처리해야 합니다.

### `recover`는 경계에서만

서버에서는 요청 경계에서 panic을 회수해 프로세스 전체 중단을 막습니다.

```go
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// stack/log/trace 기록
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

핵심:

1. recover 시 반드시 로깅/메트릭/스택 기록
2. 클라이언트에는 내부 상세를 노출하지 않음
3. panic을 숨기고 계속 진행하기보다, 원인 수정이 우선

### 라이브러리 작성 시 원칙

1. 공개 API는 가능하면 `error` 반환
2. panic 동작이 있다면 문서에 명시
3. recover로 임의 복구하지 말고 호출자 제어권을 존중

## 요약

1. Go 에러 처리의 기본은 생성 -> 래핑 -> 분기(`Is/As`) -> 경계 매핑이다.
2. try-catch 대신 `error` 반환으로 실패 흐름을 명시적으로 유지한다.
3. `panic`은 비즈니스 오류 처리에 쓰지 않는다.
4. `recover`는 서버/고루틴 경계에서 최후 방어선으로만 사용한다.

## 체크리스트

- `%w` 래핑과 `errors.Is/As`를 일관되게 사용하고 있는가
- 문자열 비교 대신 에러 타입/값 기반 분기를 하는가
- 계층별(Repository/Service/Handler) 에러 책임이 분리되어 있는가
- `panic`이 일반 실패 흐름에 섞여 있지 않은가
- recover 경계에서 로그/메트릭/스택을 남기고 있는가

## 다음 챕터

- [07. 동시성(Go의 킬러 기능): Java 스레드/Executor와 비교](./07-concurrency-vs-java.md)
