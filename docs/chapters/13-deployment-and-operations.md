# 13. 배포/운영: 실무 필수

## 빌드/크로스 컴파일

Go의 큰 장점은 단일 바이너리 배포가 쉽다는 점입니다.  
운영 안정성을 위해서는 "재현 가능한 빌드"와 "타깃 플랫폼 명시"가 핵심입니다.

### 기본 빌드

```bash
go build -o bin/api ./cmd/api
```

### 크로스 컴파일

```bash
GOOS=linux GOARCH=amd64 go build -o bin/api-linux-amd64 ./cmd/api
GOOS=linux GOARCH=arm64 go build -o bin/api-linux-arm64 ./cmd/api
```

릴리즈 파이프라인에서 멀티 아키텍처(amd64/arm64)를 함께 준비하면 배포 유연성이 높아집니다.

### 정적 바이너리 고려

가능하면 의존성을 줄인 정적 바이너리 전략이 배포/운영을 단순화합니다.

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o bin/api ./cmd/api
```

운영 체크:

1. 빌드 시 Go 버전 고정
2. 빌드 메타데이터(commit SHA, build time) 주입
3. SBOM/취약점 스캔 파이프라인 연계

`00`장에서 정한 최소 지원 Go 버전 정책도 CI에서 함께 검증해야 합니다.

## 컨테이너 베스트 프랙티스

컨테이너 이미지는 "작고, 예측 가능하고, 안전하게" 만드는 것이 원칙입니다.

### 멀티 스테이지 빌드

1. 빌드 스테이지: Go 툴체인 사용
2. 런타임 스테이지: 최소 베이스 이미지 사용

예시 개념:

```dockerfile
# 팀 정책에 맞는 최소 지원 버전(MSGV) 또는 표준 버전으로 고정
ARG GO_VERSION=1.26.3
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/api /api
USER nonroot:nonroot
ENTRYPOINT ["/api"]
```

위 예시의 `1.26.3`은 2026-05 기준 샘플 값입니다. `00`/`02`장에서 정의한 팀 버전 정책(최소 지원 버전 + 최신 안정 버전 검증)에 맞게 실제 값을 고정하세요.

### 실무 체크포인트

1. root 사용자 금지
2. 불필요 패키지/셸 제거
3. 이미지 레이어 캐시 최적화(`go.mod`/`go.sum` 선복사)
4. 이미지 스캔 자동화(CI)
5. 태그 정책: `latest` 단독 사용 금지, immutable tag 권장

### 설정 주입

설정은 이미지 bake가 아니라 런타임 환경 변수로 주입합니다.

1. dev/stage/prod 분리
2. 비밀값은 시크릿 스토어 사용
3. 설정 변경 이력 추적 가능해야 함

## 헬스체크/그레이스풀 셧다운

운영 품질의 핵심은 "장애 시 빠르게 감지하고 안전하게 종료"하는 것입니다.

### 헬스체크 분리

1. Liveness: 프로세스 생존 여부
2. Readiness: 요청 처리 준비 여부(DB/Redis/NATS 연결 상태 포함)

`go-commerce-api`는 `/livez`와 `/readyz`를 분리하고, 기존 `/healthz`는 호환용 liveness로 유지합니다. `/readyz`는 Postgres, Redis, NATS 상태를 함께 확인합니다.

### 그레이스풀 셧다운

SIGTERM 수신 시 새 요청을 막고, 진행 중 요청과 백그라운드 워커를 정리한 뒤 종료합니다.

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go func() {
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}()
```

핵심:

1. 종료 타임아웃 명시
2. DB/Redis/NATS 연결 정리
3. worker 소비 중단 + in-flight 작업 처리 정책 정의

롤링 배포 시 readiness를 먼저 내려 새 트래픽을 차단한 뒤 종료하면 손실을 줄일 수 있습니다.

## 모니터링 기본

운영에서는 "보이는 서비스"가 "안전한 서비스"입니다.  
메트릭, 로그, 트레이스는 분리된 도구가 아니라 하나의 관측성 체계로 다뤄야 합니다.

### 메트릭

최소한 다음을 수집합니다.

1. RED 지표: Rate, Errors, Duration
2. 리소스: CPU, memory, goroutine, GC pause
3. 의존성: DB/Redis/NATS 지연/에러율

### 로그

1. 구조적 로그(JSON 또는 key=value)
2. request_id/trace_id 필수
3. 민감정보 마스킹
4. 에러 로그와 비즈니스 이벤트 로그 분리

### 트레이스

1. inbound HTTP span
2. outbound(DB/Redis/NATS/HTTP) span
3. 에러/타임아웃 상태 기록

### 알람 원칙

1. 사용자 영향 지표 중심(p95 지연, 에러율)
2. 과도한 알람 노이즈 제거
3. 대응 가능한 runbook 링크 포함

관측성은 장애 후 분석 도구가 아니라, 배포 직후 회귀 탐지 도구로도 사용해야 합니다.

## 릴리즈/버전/호환성 관리

릴리즈 전략이 없으면 코드 품질과 무관하게 운영 위험이 커집니다.

### 릴리즈 방식

1. Blue/Green
2. Rolling
3. Canary

초기에는 Rolling + 빠른 롤백 기준을 먼저 정하고, 트래픽 규모가 커지면 Canary 비중을 늘리는 전략이 현실적입니다.

### 버전 정책

1. Semantic Versioning(semver) 기본
2. API 계약 변경 시 명확한 버전 분기(`/v1`, `/v2`)
3. 변경 로그(CHANGELOG)와 마이그레이션 가이드 제공

### 호환성 관리

1. DB 스키마: backward-compatible migration 우선
2. 이벤트 스키마: consumer가 구버전 필드를 허용하도록 진화
3. 설정 키: deprecated 기간을 두고 단계적으로 제거

### 배포 게이트(권장)

배포 전 자동 체크:

1. `go test ./...` / `go test -race ./...`
2. 린트 및 보안 스캔
3. 컨테이너 빌드/실행 스모크 테스트(smoke test)
4. 마이그레이션 dry-run 검증

배포 후 확인:

1. 에러율/지연 시간 변화
2. 핵심 비즈니스 지표(주문 생성 성공률 등)
3. 로그 이상 패턴

## 요약

1. 배포 품질의 핵심은 재현 가능한 빌드와 안전한 롤백 전략이다.
2. 컨테이너는 최소 권한/최소 이미지 원칙으로 운영해야 한다.
3. 헬스체크와 그레이스풀 셧다운은 무중단 배포의 기본 전제다.
4. 메트릭/로그/트레이스를 통합해 배포 회귀를 빠르게 감지해야 한다.
5. 버전/호환성 정책 없이는 서비스 확장 시 운영 비용이 급격히 증가한다.

## 체크리스트

- 빌드 아티팩트가 버전/커밋 기준으로 재현 가능한가
- 컨테이너가 non-root, 최소 이미지, 스캔 정책을 준수하는가
- readiness/liveness와 graceful shutdown이 분리 설계되어 있는가
- RED 지표와 의존성 지표에 알람이 설정되어 있는가
- 릴리즈 전/후 점검 게이트와 롤백 기준이 문서화되어 있는가

## 다음 챕터

- [14. Java -> Go 마이그레이션 가이드(선택)](./14-java-to-go-migration-guide.md)
