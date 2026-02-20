# 03. 언어 기초 문법(자바 대비 포인트 중심)

## 파일/패키지/임포트/가시성 규칙

Go는 파일보다 "패키지" 단위로 코드를 조직합니다.  
같은 디렉터리의 `.go` 파일은 보통 같은 `package`를 선언합니다.

### 기본 규칙

1. 실행 진입점은 `package main` + `func main()`.
2. 라이브러리 코드는 보통 기능 단위 패키지로 분리.
3. import는 사용한 것만 허용(미사용 import는 컴파일 에러).

Java와의 큰 차이는 가시성 규칙입니다.

- Go: 식별자 첫 글자가 대문자면 export, 소문자면 package-private
- Java: `public/private/protected` 키워드로 제어

예시:

```go
package order

type Service struct{}      // 외부 패키지에서 접근 가능
type repository struct{}   // 같은 패키지에서만 접근 가능

func NewService() *Service { // 외부 패키지에서 호출 가능
	return &Service{}
}

func validateInput() error { // 같은 패키지 내부 함수
	return nil
}
```

이 규칙 덕분에 API 표면을 줄이기 쉽습니다.  
외부에 공개할 식별자만 대문자로 두고, 나머지는 기본적으로 숨긴다고 생각하면 됩니다.

## 변수/상수/타입 추론/제로값

Go는 선언 방식이 간결하고, "초기값 없는 선언"도 안전하게 동작합니다.

### 변수 선언

```go
var count int        // zero value: 0
name := "gopher"     // 함수 내부에서만 사용 가능한 짧은 선언
var enabled = true   // 타입 추론
```

`:=`는 Java의 `var`와 비슷해 보이지만, "재할당"이 아니라 "새 변수 선언"이라는 점에 주의해야 합니다.

### 상수

```go
const appName = "go-commerce-api"
const timeoutSec = 3
```

필요하면 `iota`로 열거 상수를 만들 수 있습니다.

```go
const (
	StatusPending = iota
	StatusPaid
	StatusCanceled
)
```

### zero value

Go는 변수 선언 시 기본값이 자동 할당됩니다.

1. 숫자: `0`
2. bool: `false`
3. string: `""`
4. 포인터/슬라이스/맵/함수/인터페이스: `nil`

Java에서 `null` 체크를 많이 하던 습관이 있다면, Go에서는 zero value 설계를 먼저 고려하면 코드가 단순해집니다.

## 제어문

Go에는 `while`이 없고 `for`가 유일한 반복문입니다.

### `if`

짧은 초기화 문장을 함께 쓸 수 있습니다.

```go
if err := validate(req); err != nil {
	return err
}
```

`if` 블록 안에서만 `err`가 유효하므로 스코프 관리에 유리합니다.

### `for`

```go
for i := 0; i < 3; i++ {
	fmt.Println(i)
}

for _, item := range items {
	fmt.Println(item)
}
```

`range`는 컬렉션 순회의 기본입니다.  
인덱스가 필요 없으면 `_`로 버려 불필요한 변수 생성을 막습니다.

### `switch`

Go의 `switch`는 기본적으로 `break`가 자동입니다.

```go
switch status {
case "paid":
	return handlePaid()
case "canceled":
	return handleCanceled()
default:
	return handlePending()
}
```

### `defer`

함수 종료 직전에 실행할 정리 작업에 사용합니다.

```go
f, err := os.Open("data.txt")
if err != nil {
	return err
}
defer f.Close()
```

DB row close, unlock, span 종료 같은 정리 코드에 매우 유용합니다.  
여러 `defer`는 LIFO(나중 등록한 것이 먼저 실행) 순서로 동작합니다.

## 함수

Go 함수는 다중 반환을 기본으로 지원합니다.  
이 패턴이 에러 처리 스타일과 결합되어 Go 코드의 기본 형태를 만듭니다.

### 다중 반환

```go
func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
```

호출자는 성공 값과 에러를 동시에 받습니다.

### named return

```go
func split(full string) (first string, last string) {
	parts := strings.SplitN(full, " ", 2)
	if len(parts) == 2 {
		first, last = parts[0], parts[1]
	}
	return
}
```

named return은 편리하지만, 복잡한 함수에서는 오히려 가독성을 해칠 수 있습니다.  
짧고 의도가 분명한 경우에만 제한적으로 사용하세요.

### variadic 함수

```go
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
```

호출 시 여러 인자를 전달하거나 슬라이스를 `...`로 펼칠 수 있습니다.

```go
values := []int{1, 2, 3}
result := sum(values...)
```

## 에러 처리 스타일

Go에서 에러는 "예외(Exception)"가 아니라 "값(value)"입니다.  
핵심은 발생 지점에서 숨기지 않고 호출자에게 명시적으로 전달하는 것입니다.

### 기본 패턴

```go
user, err := repo.FindByID(ctx, id)
if err != nil {
	return nil, err
}
```

### 에러 래핑(wrap)

맥락을 추가할 때 `%w`를 사용합니다.

```go
if err != nil {
	return fmt.Errorf("find user id=%d: %w", id, err)
}
```

상위 계층에서는 `errors.Is`, `errors.As`로 원인 분기 가능합니다.

```go
if errors.Is(err, sql.ErrNoRows) {
	return ErrUserNotFound
}
```

### sentinel error vs typed error

1. Sentinel error  
   예: `var ErrNotFound = errors.New("not found")`  
   단순 비교에 유리하지만, 문맥 정보 확장이 제한적입니다.
2. Typed error  
   구조체 기반 에러 타입으로 필드(코드, 리소스 ID 등)를 담을 수 있습니다.  
   분기와 로깅에 유리하지만 타입 수가 과도하게 늘 수 있습니다.

실무에서는 다음 기준을 자주 사용합니다.

1. 단순 상태 신호: sentinel
2. 추가 메타데이터 필요: typed error
3. 경계 밖으로 전달할 때: wrap으로 맥락 보강

### `panic`의 위치

일반 비즈니스 에러 처리에 `panic`을 사용하지 않습니다.  
`panic`은 보통 "복구 불가능한 프로그래밍 오류"나 "프로세스 부팅 실패" 같은 경계 상황에 한정합니다.

## 요약

1. Go는 패키지와 export(대문자) 규칙으로 API 경계를 단순하게 만든다.
2. zero value와 짧은 선언(`:=`)을 이해하면 코드가 훨씬 간결해진다.
3. `for`/`switch`/`defer`는 Go 코드 흐름의 핵심 도구다.
4. 함수 다중 반환과 `error` 값 전달이 Go 에러 처리의 기본 패턴이다.
5. wrap, sentinel, typed error를 상황에 맞게 선택해야 한다.

## 체크리스트

- export 규칙(대문자/소문자)을 패키지 설계에 적용하고 있는가
- `:=`가 선언인지 재할당인지 구분하고 있는가
- 리소스 정리를 `defer`로 일관되게 처리하고 있는가
- 함수 반환값에서 `error`를 즉시 처리하고 있는가
- `panic`을 일반 흐름 제어에 사용하지 않고 있는가
