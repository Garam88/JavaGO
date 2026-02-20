# 10. 테스트 전략: Java(JUnit/Mockito) 관점에서

## 유닛 테스트

Go 테스트의 기본 철학은 "작고 빠른 테스트를 많이, 표준 도구로 반복 실행"입니다.  
Java의 JUnit과 달리 Go는 언어/도구 체인에 테스트가 기본 내장되어 있습니다.

### 기본 형태

```go
func TestAdd(t *testing.T) {
	got := Add(1, 2)
	want := 3
	if got != want {
		t.Fatalf("got=%d want=%d", got, want)
	}
}
```

### 테이블 드리븐 테스트

Go에서 가장 많이 쓰는 패턴입니다.

```go
func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "paid", in: "PAID", want: "paid"},
		{name: "empty", in: "", want: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeStatus(tc.in)
			if got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}
```

장점:

1. 입력/기대값 케이스를 빠르게 확장 가능
2. 실패 시 케이스 이름으로 원인 파악이 쉬움
3. 중복 테스트 코드 감소

### 실무 테스트 루틴

```bash
go test ./...
go test -race ./...
go test -cover ./...
```

`07`장에서 다룬 동시성 코드가 많다면 `-race`는 필수로 포함합니다.

## 목킹 전략

Java Mockito처럼 동적 mock 프레임워크를 먼저 찾기보다, Go에서는 작은 인터페이스 + fake/stub 구현으로 시작하는 것이 보통 더 단순합니다.

### 인터페이스 경계 설정

```go
type OrderRepository interface {
	Save(ctx context.Context, o Order) error
}
```

테스트에서는 fake를 직접 만듭니다.

```go
type fakeOrderRepo struct {
	saveErr error
	saved   []Order
}

func (f *fakeOrderRepo) Save(ctx context.Context, o Order) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, o)
	return nil
}
```

### fake/stub/spies 구분

1. Stub: 고정 응답 반환
2. Fake: 단순 동작 구현(메모리 저장소 등)
3. Spy: 호출 여부/인자 기록

작은 프로젝트에서는 fake 하나로 충분한 경우가 많습니다.

### 언제 mock 프레임워크를 고려할까

1. 외부 의존이 복잡하고 시나리오 분기가 많은 경우
2. 팀에서 이미 표준 도구를 사용 중인 경우
3. 코드 생성 기반 접근이 유지보수 가능한 경우

핵심은 도구보다 인터페이스 크기입니다. 인터페이스가 크면 어떤 방식의 목킹도 고통스럽습니다.

## 통합 테스트

`go-commerce-api`에서는 DB/Redis/MQ 연동이 핵심이므로 통합 테스트가 품질을 크게 좌우합니다.

### 테스트 계층 구분

1. Unit: 외부 의존 없이 함수/서비스 로직 검증
2. Integration: 실제 DB/Redis/MQ와의 계약 검증
3. E2E(선택): API 경로 전체 검증

### DB 통합 테스트 기본 패턴

1. 테스트 시작 전 스키마 준비(마이그레이션)
2. 각 테스트를 트랜잭션으로 감싸고 롤백
3. 테스트 데이터는 명시적으로 생성/정리

```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()
repo := NewOrderRepo(tx)
```

### 컨테이너 기반 테스트

로컬/CI 재현성을 높이기 위해 Docker 기반 테스트 환경을 자주 사용합니다.

운영 포인트:

1. 테스트 격리(테스트마다 DB 스키마/키 분리)
2. 테스트 실행 시간 관리(느린 테스트 태깅)
3. flaky 테스트 차단(재시도보다 원인 제거 우선)

### 병렬 실행 주의

`t.Parallel()`은 속도를 올리지만 공유 자원(DB, 포트, 파일)을 명확히 분리하지 않으면 불안정해집니다.

병렬화 전에 확인:

1. 테스트 데이터 충돌 여부
2. 랜덤 포트/리소스 분리 여부
3. 클린업 누락 여부

## 벤치마크

Go 벤치마크는 `testing.B` 기반으로 간단히 작성할 수 있고, 성능 회귀 감지에 매우 효과적입니다.

### 기본 벤치마크

```go
func BenchmarkNormalizeStatus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NormalizeStatus("PAID")
	}
}
```

실행:

```bash
go test -bench=. -benchmem ./...
```

`-benchmem`으로 할당 수를 함께 보면 GC 부담 증가를 조기에 감지할 수 있습니다.

### 비교 실험 패턴

1. 변경 전 기준값 측정
2. 코드 변경
3. 동일 조건 재측정
4. 차이 분석 후 채택 여부 결정

성능 최적화는 "추측"이 아니라 "측정"으로 결정합니다.

### 프로파일링과 연결

벤치마크에서 병목이 확인되면 CPU/메모리 프로파일링으로 원인을 좁힙니다.

```bash
go test -run=^$ -bench=BenchmarkNormalizeStatus -cpuprofile=cpu.out -memprofile=mem.out ./...
go tool pprof cpu.out
```

## 요약

1. Go 테스트는 표준 도구(`go test`) 중심으로 빠르게 반복하는 것이 기본이다.
2. 유닛 테스트는 테이블 드리븐 패턴이 유지보수성과 확장성이 높다.
3. 목킹은 작은 인터페이스 + fake/stub부터 시작하는 것이 실무적으로 유리하다.
4. 통합 테스트는 실제 인프라 계약을 검증하고, 격리 전략이 품질의 핵심이다.
5. 벤치마크와 프로파일링을 함께 써야 성능 회귀를 안정적으로 잡을 수 있다.

## 체크리스트

- 핵심 비즈니스 로직에 테이블 드리븐 테스트가 적용되어 있는가
- 인터페이스가 테스트 가능한 크기로 유지되는가
- 통합 테스트 환경이 로컬/CI에서 재현 가능한가
- `-race`, `-cover`, `-benchmem`를 정기 루틴에 포함했는가
- 성능 변경은 측정 결과로 의사결정하고 있는가
