# 02. 개발 환경 세팅 & 툴체인

## 설치, 버전 관리, go env

`00`장에서 정한 기준처럼 이 책은 Go 최신 안정 버전을 기본으로 설명합니다.  
핵심은 "설치 그 자체"보다 "팀이 같은 방식으로 재현 가능한 환경"입니다.

먼저 로컬 상태를 확인합니다.

```bash
go version
go env GOROOT GOPATH GOMOD GOCACHE GOOS GOARCH
```

자주 보는 항목은 다음과 같습니다.

1. `GOMOD`: 현재 디렉터리가 어떤 모듈에 속하는지
2. `GOCACHE`: 빌드/테스트 캐시 위치
3. `GOOS`, `GOARCH`: 타깃 플랫폼

Java의 SDK 관리와 비슷하게, Go도 프로젝트별 버전 고정 전략이 중요합니다.

1. 로컬: 최신 안정 버전
2. CI: 최소 지원 버전 + 최신 안정 버전 매트릭스
3. 문서: 최소 지원 Go 버전 명시

팀 온보딩 시에는 아래 두 줄부터 실행하도록 안내하면 좋습니다.

```bash
go version
go test ./...
```

## 모듈/의존성

Go의 의존성 관리는 `go.mod`/`go.sum`이 기준입니다.

### 기본 흐름

```bash
go mod init github.com/your-org/go-commerce-api
go mod tidy
go list -m all
```

실무에서 기억할 점:

1. 코드를 import한 뒤 `go test`/`go build`를 실행하면 필요한 모듈이 자동 반영됩니다.
2. `go.mod`는 의존성 선언, `go.sum`은 검증용 체크섬 역할입니다.
3. 둘은 항상 함께 커밋합니다.

### `go get`의 용도

- 라이브러리 추가/업데이트: `go get example.com/lib@vX.Y.Z`
- 툴 설치(실행 바이너리): `go install example.com/cmd/tool@latest`

라이브러리와 도구 설치를 섞지 않는 것이 중요합니다.

### `replace`와 로컬 개발

모노레포나 로컬 패키지 실험 시 `replace`를 쓸 수 있습니다.

```go
replace github.com/your-org/shared => ../shared
```

주의:

1. 로컬 경로 `replace`는 팀/CI에서 깨질 수 있습니다.
2. 임시 실험이 끝나면 제거하고 원래 모듈 버전으로 되돌립니다.

### 프록시와 사설 모듈

네트워크 정책이 있는 조직에서는 아래 환경을 함께 관리합니다.

- `GOPROXY`
- `GOSUMDB`
- `GOPRIVATE`

예시:

```bash
go env -w GOPRIVATE=github.com/your-org/*
```

이 설정이 없으면 사설 저장소 의존성 다운로드에서 자주 막힙니다.

## 포매팅/린팅

Go의 장점 중 하나는 포맷 규칙이 사실상 표준으로 고정된다는 점입니다.  
"팀 컨벤션 논쟁"보다 "자동 정리"에 집중하면 됩니다.

### 최소 기준

1. `gofmt`: 코드 포맷
2. `goimports`: 포맷 + import 정리
3. `golangci-lint`: 다중 린터 집합 실행

자주 쓰는 명령:

```bash
gofmt -w .
goimports -w .
golangci-lint run ./...
```

실무 권장:

1. 로컬 저장 전 `goimports`
2. PR 전 `golangci-lint`
3. CI에서 동일 명령 강제

Java에서 Checkstyle/SpotBugs/PMD를 CI로 강제하는 패턴과 유사하다고 보면 됩니다.

## 테스트/벤치마크/프로파일링

이 책의 학습 루프(개념 -> 코드 -> 확장 -> 검증)에서 검증 단계의 기본 명령은 아래입니다.

### 테스트

```bash
go test ./...
go test -race ./...
go test -cover ./...
```

설명:

- `-race`: 데이터 경쟁 탐지
- `-cover`: 패키지 커버리지 확인

### 벤치마크

```bash
go test -bench=. -benchmem ./...
```

`-benchmem`을 함께 보면 할당 수/바이트를 추적할 수 있어 성능 회귀를 잡기 쉽습니다.

### 프로파일링(pprof)

테스트 함수에 CPU/메모리 프로파일을 연결해 분석합니다.

```bash
go test -run=^$ -bench=. -cpuprofile=cpu.out -memprofile=mem.out ./...
go tool pprof cpu.out
```

초기에는 다음 순서로 접근하면 충분합니다.

1. 느린 지점을 벤치마크로 재현
2. CPU 프로파일로 핫스팟 확인
3. 메모리 프로파일로 불필요 할당 확인
4. 수정 후 벤치마크 재측정

측정 없이 최적화부터 시작하지 않는 것이 핵심입니다.

## 문서화/패키지 검색

Go 문서는 코드와 매우 밀접합니다.  
공식 문서를 읽을 때는 "설명 -> 예제 -> 시그니처 -> zero value/동시성 주의사항" 순서로 보면 빠릅니다.

### `pkg.go.dev` 읽는 순서

1. 패키지 개요(Overview)
2. 예제(Examples)
3. 타입/함수 시그니처
4. Deprecated/Notes

특히 표준 라이브러리는 예제가 품질 높은 경우가 많아, 먼저 복사해 실행해 보는 것이 좋습니다.

### 로컬 문서 확인

```bash
go doc net/http
go doc net/http.HandlerFunc
```

IDE에서도 hover/definition으로 거의 같은 정보를 바로 확인할 수 있습니다.

### 패키지 선택 기준(짧은 체크)

외부 라이브러리를 도입할 때는 아래를 확인합니다.

1. 유지보수 상태(최근 릴리스/이슈 응답)
2. API 안정성(메이저 버전 정책)
3. 라이선스 적합성
4. 대체 가능성(표준 라이브러리로 충분한가)

## 요약

1. 환경 세팅의 핵심은 "재현성"이다.
2. 의존성은 `go.mod`/`go.sum`으로 추적하고 함께 관리한다.
3. 포맷/린트/테스트는 로컬과 CI에서 같은 규칙으로 강제한다.
4. 성능 튜닝은 벤치마크와 프로파일링 기반으로 진행한다.
5. 문서는 `pkg.go.dev` 예제 중심으로 읽고 바로 실행해 검증한다.

## 체크리스트

- `go version`, `go env`로 현재 환경을 설명할 수 있는가
- `go mod init/tidy`와 `replace` 사용 시 주의점을 알고 있는가
- `goimports`/`golangci-lint`를 로컬 루틴에 포함했는가
- `go test -race`, `go test -bench`를 주기적으로 실행하는가
- 외부 라이브러리 도입 시 유지보수/안정성 기준을 확인하는가
