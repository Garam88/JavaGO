# 04. 타입 시스템: Java 개발자가 헷갈리는 지점 정리

## 기본 타입과 변환

Go는 "명시적 변환"을 강하게 요구합니다.  
Java처럼 자동 승격/변환에 기대기보다, 어떤 타입이 오가는지 코드에 드러내는 것이 기본입니다.

### 기본 타입 감각

1. 정수: `int`, `int64`, `uint` 등 플랫폼/용도별 구분
2. 실수: `float32`, `float64`
3. 불리언: `bool`
4. 문자열: `string` (immutable)

실무에서는 외부 경계(DB, JSON, 네트워크)에서 `int`보다 명시적인 `int64`를 더 자주 씁니다.

### 명시적 변환

```go
var n int32 = 10
var m int64 = int64(n)
```

Go는 축소 변환 시 데이터 손실 가능성을 자동으로 막아주지 않습니다.  
범위가 중요한 변환은 사전에 값 검증을 넣어야 합니다.

### 문자열, 바이트, 룬

이 부분은 Java 개발자가 가장 자주 헷갈립니다.

1. `string`: UTF-8 바이트 시퀀스
2. `[]byte`: 가변 바이트 배열
3. `rune`: 유니코드 코드포인트(`int32` 별칭)

```go
s := "한a"
fmt.Println(len(s))         // 바이트 길이
fmt.Println([]rune(s))      // 룬 단위 변환
fmt.Println(len([]rune(s))) // 문자(코드포인트) 길이
```

요점:

1. `len(string)`은 문자 수가 아니라 바이트 수입니다.
2. 문자 단위 처리(자르기/인덱싱)는 `[]rune` 변환을 고려해야 합니다.
3. 문자열 빌드는 반복 `+`보다 `strings.Builder`가 안전하고 효율적입니다.

## 구조체와 메서드, 임베딩

Go에서 `struct`는 데이터 묶음이고, 동작은 receiver 메서드로 붙입니다.

```go
type Order struct {
	ID     int64
	Status string
}

func (o Order) IsPaid() bool {
	return o.Status == "paid"
}
```

Java의 클래스와 비슷해 보이지만 상속이 없고, 조합(embedding)을 선호합니다.

### value receiver vs pointer receiver

```go
func (o Order) Snapshot() string { // 값 복사
	return o.Status
}

func (o *Order) MarkPaid() { // 원본 변경
	o.Status = "paid"
}
```

선택 기준:

1. 상태 변경 필요: pointer receiver
2. 큰 구조체 복사 비용 회피: pointer receiver
3. 불변 성격의 읽기 전용 동작: value receiver 가능

한 타입에서 receiver 스타일은 일관되게 유지하는 것이 좋습니다.

### 임베딩(embedding)

```go
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Product struct {
	ID   int64
	Name string
	Timestamps
}
```

`Product.CreatedAt`처럼 승격된 필드 접근이 가능합니다.  
상속처럼 쓰기보다, 재사용 가능한 공통 필드를 조합하는 용도로 제한하는 것이 안전합니다.

## 포인터 개념

Go 포인터는 Java 참조와 비슷한 목적(공유/변경)을 갖지만 훨씬 명시적입니다.

### 언제 쓰는가

1. 함수에서 원본 상태를 변경해야 할 때
2. 큰 구조체 복사 비용을 줄이고 싶을 때
3. `nil`로 "값 없음"을 표현해야 할 때(선택 필드)

### 언제 피하는가

1. 작은 값 타입을 굳이 공유할 이유가 없을 때
2. 단순 계산 함수에서 불필요한 간접 참조가 생길 때
3. 생명주기와 소유권을 더 헷갈리게 만들 때

예시:

```go
type User struct {
	Nickname *string // 선택값
}
```

값이 꼭 존재하는 필드는 포인터보다 값 타입이 보통 더 단순합니다.

추가로 기억할 점:

1. Go는 포인터 산술을 허용하지 않습니다(안전성).
2. `new(T)`는 `*T`를 만들지만, 실무에서는 리터럴(`&T{}`)이 더 자주 쓰입니다.

## 슬라이스/배열/맵

이 섹션은 성능과 버그 양쪽에서 매우 중요합니다.

### 배열(array)

```go
var a [3]int
```

배열은 길이가 타입의 일부입니다.  
`[3]int`와 `[4]int`는 다른 타입입니다.

### 슬라이스(slice)

```go
s := []int{1, 2, 3}
s = append(s, 4)
```

슬라이스는 내부적으로 `(포인터, 길이, 용량)`을 가진 뷰입니다.  
즉, 복사해도 같은 backing array를 공유할 수 있습니다.

```go
base := []int{1, 2, 3, 4}
x := base[:2]
y := base[:2]
y[0] = 99
fmt.Println(x[0]) // 99 (같은 배열 공유)
```

공유를 끊고 싶으면 복사합니다.

```go
copied := append([]int(nil), base...)
```

### 맵(map)

```go
m := map[string]int{"apple": 3}
v, ok := m["apple"]
```

핵심 주의점:

1. 맵은 참조 성격이라 함수에 넘기면 내부 변경이 반영됩니다.
2. 존재 여부는 항상 `v, ok` 패턴으로 확인합니다.
3. 맵은 동시 쓰기에 안전하지 않습니다. 동시 접근은 `sync.Mutex` 또는 `sync.Map` 등 별도 전략이 필요합니다.

### 성능 감각

1. 슬라이스는 예상 크기를 알면 `make([]T, 0, n)`으로 용량을 미리 할당
2. 큰 객체를 자주 append/delete할 때 할당 추적(`-benchmem`) 확인
3. 맵 키 타입은 비교 비용을 고려해 선택(긴 문자열 키 남용 주의)

## 인터페이스

Go 인터페이스는 "명시적 선언 없이" 메서드 집합이 맞으면 구현으로 간주됩니다.

```go
type Notifier interface {
	Notify(msg string) error
}

type SlackNotifier struct{}

func (s SlackNotifier) Notify(msg string) error {
	return nil
}
```

`SlackNotifier`가 `Notifier`를 구현한다고 따로 선언하지 않아도 됩니다.

### 작은 인터페이스 원칙

Java처럼 큰 계약 인터페이스를 먼저 만들기보다, 사용 지점이 요구하는 최소 메서드만 정의합니다.

```go
type Clock interface {
	Now() time.Time
}
```

테스트 대역(fakes/stubs) 작성이 쉬워지고 결합도가 낮아집니다.

### nil interface 함정

가장 흔한 버그 중 하나입니다.

```go
var p *MyError = nil
var err error = p
fmt.Println(err == nil) // false
```

이유: 인터페이스 값은 `(동적 타입, 동적 값)` 쌍으로 표현되며, 타입 정보가 있으면 `nil` 비교가 false가 됩니다.

예방 규칙:

1. 반환 직전 `if err != nil { ... }` 패턴을 일관되게 유지
2. `*CustomError`를 `error`로 다루는 경계에서 특히 주의
3. 필요하면 `errors.As`로 타입 분기

## 제네릭

Go 제네릭은 강력하지만, 무분별하게 쓰면 오히려 가독성을 떨어뜨립니다.  
이 책에서는 "중복이 명확할 때만" 도입하는 기준을 권장합니다.

### 기본 문법

```go
func First[T any](items []T) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}
	return items[0], true
}
```

`T any`는 모든 타입 허용을 의미합니다.

### 제약(constraint)

```go
func Contains[T comparable](items []T, target T) bool {
	for _, v := range items {
		if v == target {
			return true
		}
	}
	return false
}
```

`comparable`은 `==` 비교가 가능한 타입만 허용합니다.

### 실무 적용 패턴

1. 컬렉션 유틸(contains, map/filter 계열)
2. 반복되는 DTO 변환기
3. 자료구조(큐/셋) 공통 구현

다음 상황에서는 제네릭보다 일반 코드를 우선합니다.

1. 사용처가 1~2곳뿐인 경우
2. 제약이 복잡해 시그니처 해석 비용이 큰 경우
3. 런타임 다형성(interface)으로 충분한 경우

핵심은 "중복 제거"보다 "코드 이해 비용"까지 함께 비교하는 것입니다.

## 요약

1. Go 타입 변환은 명시적이며, 문자열/바이트/룬 구분이 중요하다.
2. `struct` + receiver + embedding 조합으로 모델을 설계한다.
3. 포인터는 변경/공유/선택값 표현에만 필요한 만큼 사용한다.
4. 슬라이스 공유 메모리와 맵 동시성 제약을 정확히 이해해야 한다.
5. 인터페이스는 작게 정의하고, nil interface 함정을 피해야 한다.
6. 제네릭은 중복과 가독성을 함께 비교해 도입한다.

## 체크리스트

- `len(string)`과 `len([]rune(string))` 차이를 설명할 수 있는가
- receiver를 value/pointer 중 하나로 일관되게 선택하고 있는가
- 슬라이스 공유로 인한 사이드이펙트를 인지하고 있는가
- 맵 동시 쓰기 위험을 회피하는 전략을 갖고 있는가
- `nil interface` 함정을 이해하고 `error` 비교를 안전하게 작성하는가
- 제네릭 도입 시 읽기 난이도와 유지보수 비용을 같이 평가하는가
