# 05. 컬렉션/문자열/유틸 실전

## 슬라이스 패턴

슬라이스는 Go에서 가장 자주 쓰는 컬렉션입니다.  
핵심은 "append는 편하지만 backing array 공유와 재할당을 이해해야 안전하다"입니다.

### 확장(append)과 용량(capacity)

```go
items := make([]int, 0, 4)
items = append(items, 1, 2, 3)
```

예상 길이를 알면 `make([]T, 0, n)`으로 용량을 미리 잡아 불필요한 재할당을 줄일 수 있습니다.

### 복사(copy)

슬라이스 복사 없이 참조만 넘기면 원본이 함께 바뀔 수 있습니다.

```go
src := []int{1, 2, 3}
dst := append([]int(nil), src...) // 독립 복사
```

또는 `copy`를 사용합니다.

```go
dst := make([]int, len(src))
copy(dst, src)
```

### 삭제 패턴

인덱스 `i` 원소 삭제:

```go
s = append(s[:i], s[i+1:]...)
```

순서가 중요하지 않고 성능을 우선할 때는 마지막 원소 스왑이 더 효율적입니다.

```go
s[i] = s[len(s)-1]
s = s[:len(s)-1]
```

### 필터 패턴(할당 최소화)

```go
out := s[:0]
for _, v := range s {
	if keep(v) {
		out = append(out, v)
	}
}
s = out
```

기존 배열을 재사용하므로 대량 처리에서 유리합니다.

## 맵 사용 시 주의점

맵은 키-값 조회가 빠르지만, 동시성 안전성과 zero value 동작을 정확히 알아야 합니다.

### 초기화

```go
m := make(map[string]int)
// 또는
m2 := map[string]int{"apple": 3}
```

`var m map[string]int` 상태는 `nil map`이며 읽기는 가능하지만 쓰기는 panic입니다.

```go
var m map[string]int
fmt.Println(m["x"]) // 0
// m["x"] = 1       // panic
```

### 조회

존재 여부는 `value, ok` 패턴으로 확인합니다.

```go
v, ok := m["apple"]
if !ok {
	// key 없음
}
```

### 동시 접근 주의

맵은 동시 쓰기(concurrent write)에 안전하지 않습니다.  
다중 고루틴에서 접근하면 `sync.Mutex`나 `sync.RWMutex`로 보호해야 합니다.

```go
type Counter struct {
	mu sync.RWMutex
	m  map[string]int
}
```

읽기 많고 쓰기 적은 경우 `RWMutex`, 패턴이 단순한 경우 `sync.Map`도 선택지입니다.

## 문자열 처리

문자열은 immutable이므로 반복 `+`는 누적 할당을 만들기 쉽습니다.

### `strings.Builder`

```go
var b strings.Builder
b.Grow(128) // 대략 크기를 알면 선할당
b.WriteString("order=")
b.WriteString(orderID)
msg := b.String()
```

로그/응답 조립처럼 반복 연결이 많은 코드에서 효과가 큽니다.

### 포맷팅 선택

1. 디버그/가독성 우선: `fmt.Sprintf`
2. 성능 민감 경로: `strconv` + `Builder`

예시:

```go
idStr := strconv.FormatInt(id, 10)
```

### 자주 쓰는 문자열 유틸

1. 분리: `strings.Split`, `SplitN`
2. 정리: `TrimSpace`, `ToLower`
3. 포함/접두/접미: `Contains`, `HasPrefix`, `HasSuffix`

요청 파라미터 정규화, 헤더 처리, 라우팅 분기에서 자주 사용됩니다.

## JSON 직렬화/역직렬화

Go 표준 JSON은 `encoding/json`으로 대부분 처리 가능합니다.  
핵심은 struct tag와 zero value/optional 필드를 어떻게 다루느냐입니다.

### 기본 직렬화/역직렬화

```go
type CreateOrderRequest struct {
	UserID   int64  `json:"user_id"`
	Coupon   string `json:"coupon,omitempty"`
	Quantity int    `json:"quantity"`
}
```

- `json:"user_id"`: 필드명 매핑
- `omitempty`: zero value면 출력 생략

### Optional 필드 표현

요청에서 "값 없음"과 "빈 값"을 구분해야 하면 포인터를 사용합니다.

```go
type PatchUserRequest struct {
	Nickname *string `json:"nickname"`
}
```

`nil`이면 미전달, `""`이면 빈 문자열로 명확히 구분됩니다.

### 안전한 디코딩 패턴

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&req); err != nil {
	return err
}
```

알 수 없는 필드를 차단하면 API 계약이 예상치 않게 넓어지는 것을 막을 수 있습니다.

### 숫자/시간 필드 주의

1. JavaScript 클라이언트와 큰 정수 교환 시 정밀도 이슈 점검
2. 시간은 RFC3339 문자열로 통일하는 정책 권장
3. 공용 API는 응답 스키마 버전 관리 고려

## 요약

1. 슬라이스는 append/복사/삭제 패턴을 정확히 알아야 안전하다.
2. 맵은 `nil` 상태와 동시 접근 제약을 이해해야 한다.
3. 문자열 조립은 `strings.Builder`와 `strconv`를 적절히 선택한다.
4. JSON은 태그, optional 표현, 엄격 디코딩 정책이 핵심이다.

## 체크리스트

- 슬라이스를 넘길 때 공유/복사 의도를 구분하고 있는가
- 맵 조회에서 `v, ok` 패턴을 일관되게 쓰는가
- 동시 접근 맵을 락 없이 쓰지 않는가
- 문자열 반복 연결 경로에서 `Builder`를 검토했는가
- JSON 요청 디코딩 시 알 수 없는 필드 처리 정책이 있는가

## 다음 챕터

- [06. 예외가 없는 세계: 에러 처리/리커버리 패턴](./06-error-handling-and-recovery.md)
