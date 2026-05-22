package knowledge

// Topics from Effective Go and community best-practices — appended to the
// same Concepts slice via init() so they share the search index but stay
// editable as a separate file. These are wider-scope than gobyexample
// snippets: they cover IDIOMS and DESIGN RULES, not just syntax.
func init() {
	Concepts = append(Concepts, effectiveGoTopics...)
}

var effectiveGoTopics = []Concept{
	{
		Name:        "naming",
		DisplayName: "Naming в Go",
		Description: "Имена в Go короткие, по контексту. Пакет — одно слово в нижнем регистре (utf8, strconv). Экспортируемые имена — с большой буквы (Reader, Marshal). Внутри пакета — короткие (i, buf, w), вне — описательные. Геттеры БЕЗ Get-префикса: owner вместо GetOwner. Интерфейсы из одного метода — с суффиксом -er (Reader, Writer).",
		Example: `// ХОРОШО
type Reader interface {
    Read(p []byte) (n int, err error)
}
type Server struct {
    addr string  // unexported — короткое имя
    log  Logger  // тоже короткое внутри пакета
}
func (s *Server) Addr() string { return s.addr } // геттер без Get

// ПЛОХО
type ReaderInterface interface {  // "Interface" в имени — лишнее
    ReadFromStream(buffer []byte) (numberOfBytesRead int, errorThatOccurred error)
}
type HTTPServer struct {
    serverAddress string
}
func (s *HTTPServer) GetServerAddress() string { return s.serverAddress }`,
		Concepts: []string{"naming", "именование", "идиомы имён", "naming convention", "geter", "exported", "unexported", "-er suffix"},
	},
	{
		Name:        "small_interfaces",
		DisplayName: "Маленькие интерфейсы",
		Description: "Главная идиома Go — интерфейс из 1-2 методов. Чем меньше интерфейс, тем больше реализаций ему подходят. io.Reader, io.Writer, io.Closer — все по одному методу. Не делай большой интерфейс заранее: 'the bigger the interface, the weaker the abstraction'.",
		Example: `// ХОРОШО — крошечный интерфейс из stdlib
type Reader interface {
    Read(p []byte) (n int, err error)
}

// функция принимает что угодно умеющее читать: файл, сетевое соединение, bytes.Buffer
func CopyToString(r io.Reader) (string, error) {
    data, err := io.ReadAll(r)
    return string(data), err
}

// ПЛОХО — толстый интерфейс
type Storage interface {
    Connect() error
    Disconnect() error
    Read(key string) ([]byte, error)
    Write(key string, value []byte) error
    Delete(key string) error
    Migrate(version int) error
    Backup(path string) error
    // ...ещё 10 методов
}
// Под Storage может реализоваться ровно один тип — это уже не абстракция.`,
		Concepts: []string{"маленькие интерфейсы", "small interfaces", "interface design", "io.Reader", "io.Writer", "-er suffix"},
	},
	{
		Name:        "accept_interfaces_return_structs",
		DisplayName: "Принимай интерфейсы, возвращай структуры",
		Description: "Идиома Go: функции/методы должны ПРИНИМАТЬ интерфейс (расширяемость на стороне вызова) и ВОЗВРАЩАТЬ конкретный тип (полная информация на стороне вызывающего). Возвращение интерфейса прячет полезные методы и усложняет тестирование.",
		Example: `// ХОРОШО
type Store interface {
    Get(id string) (*User, error)
}

func NewService(store Store, log *slog.Logger) *Service {  // принимает Store
    return &Service{store: store, log: log}              // возвращает *Service
}

// ПЛОХО — возврат интерфейса
func NewService(store Store) ServiceInterface {  // абоненту нужно угадывать какие методы доступны
    return &Service{store: store}
}`,
		Concepts: []string{"accept interfaces", "return structs", "принимай интерфейсы", "возвращай структуры", "Go idiom", "API design"},
	},
	{
		Name:        "error_wrapping",
		DisplayName: "Обёртка ошибок",
		Description: "С Go 1.13 ошибки заворачивают через fmt.Errorf с глаголом %w. Это сохраняет оригинальную ошибку в цепочке — errors.Is/As могут её достать. Каждый уровень добавляет контекст: 'что я делал когда оно сломалось', а не пересоздаёт текст.",
		Example: `package main

import (
    "errors"
    "fmt"
    "os"
)

func loadUser(id string) (*User, error) {
    data, err := os.ReadFile("users/" + id + ".json")
    if err != nil {
        return nil, fmt.Errorf("loadUser %s: %w", id, err)  // %w оборачивает
    }
    // ... parse ...
    return user, nil
}

func main() {
    _, err := loadUser("bob")
    if err != nil {
        // распечатывает полную цепочку: "loadUser bob: open ...: no such file"
        fmt.Println(err)

        // проверка типа в цепочке
        if errors.Is(err, os.ErrNotExist) {
            fmt.Println("пользователь не найден, создаём нового")
        }
    }
}

// ПЛОХО — теряется оригинальная ошибка
return nil, fmt.Errorf("loadUser %s: %v", id, err)  // %v вместо %w`,
		Concepts: []string{"error wrapping", "обёртка ошибок", "fmt.Errorf", "%w", "errors.Is", "errors.As", "error chain", "цепочка ошибок"},
	},
	{
		Name:        "share_by_communicating",
		DisplayName: "Не дели память — общайся",
		Description: "Знаменитая мантра Go: 'Do not communicate by sharing memory; instead, share memory by communicating'. Между горутинами лучше гонять данные через канал, чем брать общий мьютекс. Mutex — для защиты простого state, каналы — для координации работы. На практике используют оба, но canal — первый выбор.",
		Example: `package main

import "fmt"

// ХОРОШО — общение через канал
func main() {
    jobs := make(chan int, 5)
    results := make(chan int, 5)

    // 3 воркера
    for w := 1; w <= 3; w++ {
        go func(id int) {
            for j := range jobs {
                results <- j * 2
            }
        }(w)
    }

    // раздаём работу
    for j := 1; j <= 5; j++ {
        jobs <- j
    }
    close(jobs)

    for i := 0; i < 5; i++ {
        fmt.Println(<-results)
    }
}`,
		Concepts: []string{"don't share, communicate", "share by communicating", "channels vs mutex", "go concurrency", "worker pool", "пул воркеров"},
	},
	{
		Name:        "context_cancellation",
		DisplayName: "context.Context",
		Description: "context.Context — стандартный механизм отмены и таймаутов через границы API. Любая функция, делающая I/O, должна принимать ctx первым параметром. context.WithTimeout / WithCancel порождают дочерние контексты — отмена родителя автоматически отменяет всех детей. Это основа graceful shutdown.",
		Example: `package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

func fetchURL(ctx context.Context, url string) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)  // отменится если ctx истёк
    if err != nil {
        return "", fmt.Errorf("fetch %s: %w", url, err)
    }
    defer resp.Body.Close()
    // ... read body ...
    return "", nil
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()  // важно! утечка context = утечка ресурсов

    _, err := fetchURL(ctx, "https://example.com")
    if err != nil {
        fmt.Println(err)  // напечатает "context deadline exceeded" если медленно
    }
}`,
		Concepts: []string{"context", "Context", "контекст", "отмена", "cancellation", "timeout", "таймаут", "graceful shutdown", "WithTimeout", "WithCancel"},
	},
	{
		Name:        "table_driven_tests",
		DisplayName: "Table-driven tests",
		Description: "Идиоматичный способ писать тесты в Go: список case-структур и цикл по ним с t.Run для подтестов. Каждый case — одна строка, легко добавлять новые. t.Parallel() параллелит подтесты. Это лучший паттерн для проверки функций с разными входами.",
		Example: `package mymath_test

import "testing"

func TestDivide(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name    string
        a, b    int
        want    int
        wantErr bool
    }{
        {"positive", 10, 2, 5, false},
        {"negative", -10, 2, -5, false},
        {"zero divisor", 10, 0, 0, true},
        {"zero numerator", 0, 5, 0, false},
    }
    for _, tc := range cases {
        tc := tc  // capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            got, err := Divide(tc.a, tc.b)
            if (err != nil) != tc.wantErr {
                t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
            }
            if got != tc.want {
                t.Errorf("Divide(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
            }
        })
    }
}`,
		Concepts: []string{"table-driven", "table tests", "табличные тесты", "subtests", "t.Run", "t.Parallel", "test cases"},
	},
	{
		Name:        "goroutine_leaks",
		DisplayName: "Утечки горутин",
		Description: "Горутина живёт пока её функция не завершится. Если она ждёт на закрытом канале без сигнала остановки — утечка. Правило: каждая горутина должна иметь чёткий путь к завершению (закрытие канала, context.Done, отдельный quit-канал). Утечка незаметна сразу, но накопится за дни работы.",
		Example: `package main

import (
    "context"
    "fmt"
    "time"
)

// ПЛОХО — горутина может застрять навсегда
func leakyWorker(jobs chan int) {
    for j := range jobs {  // ждёт пока канал не закроется
        fmt.Println(j)
    }
    // если jobs не закрыли — горутина живёт вечно
}

// ХОРОШО — context для гарантированной отмены
func goodWorker(ctx context.Context, jobs <-chan int) {
    for {
        select {
        case j, ok := <-jobs:
            if !ok {
                return  // канал закрыт
            }
            fmt.Println(j)
        case <-ctx.Done():
            return  // отмена через контекст
        }
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    jobs := make(chan int)
    go goodWorker(ctx, jobs)
    // ... через 1 секунду cancel() остановит горутину ...
}`,
		Concepts: []string{"goroutine leak", "утечка горутин", "leak", "graceful shutdown", "горутина не завершается", "вечная горутина"},
	},
}
