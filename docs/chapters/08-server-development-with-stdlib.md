# 08. 표준 라이브러리로 만드는 서버 개발(웹 백엔드)

## net/http 기본

Go 표준 라이브러리만으로도 운영 가능한 HTTP 서버를 만들 수 있습니다.  
핵심은 핸들러를 얇게 유지하고, 비즈니스 로직은 서비스 계층으로 분리하는 것입니다.

### 기본 서버 구성

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /healthz", healthHandler)
mux.HandleFunc("POST /orders", createOrderHandler)

srv := &http.Server{
	Addr:         ":8080",
	Handler:      mux,
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 10 * time.Second,
	IdleTimeout:  60 * time.Second,
}
```

운영 관점에서는 `http.ListenAndServe` 직접 호출보다 `http.Server`를 명시적으로 구성하는 편이 안전합니다.

### 핸들러 설계 원칙

1. 요청 파싱/검증
2. 서비스 호출
3. 에러 매핑 및 응답 직렬화

핸들러 내부에서 DB 접근까지 모두 처리하면 테스트와 유지보수가 빠르게 어려워집니다.

### 미들웨어 패턴

```go
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s latency=%s", r.Method, r.URL.Path, time.Since(start))
	})
}
```

공통 횡단 관심사(로그, 복구, 인증, 트레이싱)는 미들웨어 체인으로 분리합니다.

## 라우팅 전략

Go 1.22+의 `ServeMux`는 메서드 기반 패턴(`"GET /path"`)을 지원해 단순 API는 표준만으로 충분히 구현 가능합니다.

### 표준 라우팅으로 가능한 범위

1. REST 기본 CRUD
2. 정적 경로 + 경로 변수 일부
3. 미들웨어 체인

### 외부 라우터를 고려할 시점

1. 복잡한 경로 매칭이 많을 때
2. 라우트 그룹/버전 관리가 대규모일 때
3. 팀 내 이미 표준화된 프레임워크가 있을 때

선택 기준:

1. 기능 요구가 정말 필요한가
2. 학습/운영 비용이 수용 가능한가
3. 종속(lock-in) 없이 교체 가능한가

초기 `go-commerce-api` 단계에서는 `ServeMux`로 시작하고, 실제 제약이 드러날 때 확장하는 전략이 안정적입니다.

## 요청/응답 모델

요청/응답 모델은 "계약 안정성"이 핵심입니다.  
핸들러는 transport DTO와 도메인 모델을 분리해 관리하는 편이 좋습니다.

### JSON 요청 처리 기본

```go
var req CreateOrderRequest
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&req); err != nil {
	writeError(w, http.StatusBadRequest, "invalid request")
	return
}
```

`DisallowUnknownFields()`를 사용하면 클라이언트 오타나 계약 이탈을 초기에 막을 수 있습니다.

### 응답 모델

```go
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
```

응답 유틸을 공통화하면 상태 코드/헤더 처리 누락을 줄일 수 있습니다.

### 스트리밍

대량 데이터나 진행 상황 전달에는 chunked response/streaming이 유용합니다.

1. `http.Flusher` 지원 여부 확인
2. 중간 flush로 지연 완화
3. 클라이언트 disconnect(`r.Context().Done()`) 감지

### 파일 업로드

```go
const maxUploadSize = 10 << 20 // 10MB
r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize) // 요청 본문 전체 제한
if err := r.ParseMultipartForm(10 << 20); err != nil { // 메모리 버퍼 상한(초과분은 임시 파일 사용)
	writeError(w, http.StatusBadRequest, "invalid multipart")
	return
}
file, header, err := r.FormFile("file")
```

안전 포인트:

1. 요청 본문 전체 크기 제한(`http.MaxBytesReader`)
2. `ParseMultipartForm` 메모리 버퍼 상한 설정
3. 파일명/확장자 신뢰 금지
4. MIME/type 검증 및 저장 경로 고정
5. 악성 파일 검사 정책 연계

## 인증/인가 개요

인증(authentication)과 인가(authorization)는 분리해 생각해야 합니다.

1. 인증: "누구인지" 확인
2. 인가: "무엇을 할 수 있는지" 판단

### JWT 기반 접근(무상태)

장점:

1. 서버 세션 저장 부담이 작음
2. 서비스 간 전달이 쉬움

주의:

1. 만료 시간 짧게 설정
2. 서명 키 회전 전략 필요
3. 클레임 최소화(민감 정보 금지)

### 세션 기반 접근(상태 저장)

장점:

1. 중앙 무효화/강제 로그아웃이 쉬움
2. 권한 변경 반영이 단순

주의:

1. 세션 저장소 운영 필요
2. 수평 확장 시 저장소 일관성 고려

### 미들웨어 배치 예시

1. Recover
2. Request ID/Tracing
3. AuthN
4. AuthZ
5. Handler

인증 실패(401)와 권한 없음(403)을 명확히 구분해야 클라이언트 동작과 운영 분석이 쉬워집니다.

## 로깅/트레이싱 기본

`go-commerce-api` 같은 서비스에서는 "문제가 생겼을 때 원인 추적 가능"이 목표입니다.

### 구조적 로그

문장 로그보다 key-value 로그를 권장합니다.

```go
log.Printf("level=info msg=\"order created\" order_id=%d user_id=%d", orderID, userID)
```

필수 필드 예시:

1. timestamp
2. level
3. request_id/trace_id
4. route/method/status
5. latency

### 요청 단위 상관관계

요청 시작 시 `request_id`를 생성하거나 전달받고, 로그/응답 헤더에 함께 넣습니다.

1. 로그 탐색 속도 향상
2. 장애 구간 상관관계 분석 가능

### 트레이싱 기본 전략

1. inbound HTTP에서 span 시작
2. outbound(DB/Redis/MQ/HTTP) span 연결
3. 에러 발생 시 span status 기록

이 장에서는 개념을 정리하고, 실제 계측 구현은 운영 장(13장)에서 확장합니다.

## 요약

1. `net/http`만으로도 운영 가능한 백엔드 서버를 구성할 수 있다.
2. 핸들러는 얇게, 공통 관심사는 미들웨어로 분리한다.
3. 라우팅은 표준으로 시작하고 복잡도 근거가 생길 때 외부 라우터를 도입한다.
4. 요청/응답 계약은 엄격 디코딩과 공통 응답 유틸로 안정화한다.
5. 인증/인가와 관측성(로그/트레이싱)은 초기부터 구조화해 두는 것이 유리하다.

## 체크리스트

- `http.Server` 타임아웃을 명시적으로 설정했는가
- 핸들러에서 비즈니스 로직을 서비스 계층으로 분리했는가
- JSON 디코딩 시 unknown field(알 수 없는 필드) 정책을 적용했는가
- 인증(401)과 인가(403)를 구분해 처리하는가
- 요청 단위 `request_id`/`trace_id`를 로그에 남기고 있는가

## 다음 챕터

- [09. 데이터 계층: DB/캐시/메시징](./09-data-layer-db-cache-messaging.md)
