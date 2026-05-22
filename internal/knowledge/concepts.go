package knowledge

// Concept is a general Go feature or stdlib pattern — distinct from a
// Piscine exercise. Where Exercise represents "the canonical answer to
// this specific 01edu task", Concept represents "a self-contained mini-
// tutorial on this Go feature": what it is, when you use it, and the
// shortest working example.
//
// Both types feed the same Search; the AI handler renders them with
// slightly different headers so the model knows whether it's looking at
// a reference solution or a language explainer.
type Concept struct {
	Name        string   // canonical lowercase key, e.g. "goroutines"
	DisplayName string   // shown in the prompt, e.g. "Goroutines"
	Description string   // one paragraph in Russian
	Example     string   // minimal working Go snippet (no package piscine wrapper)
	Concepts    []string // search keywords (Russian + English)
}

// Concepts is the curated set of Go language and stdlib mini-tutorials.
// Selection is biased toward what students at 01.tomorrow-school.ai
// actually run into during Piscine and the first real-project month:
// the basics, the pointer/slice/map troika, concurrency, and the
// idioms grade automation tools tend to enforce.
var Concepts = []Concept{
	// ─── Basics ─────────────────────────────────────────────────────────
	{
		Name:        "variables",
		DisplayName: "Переменные",
		Description: "Go — статически типизированный язык. Переменную объявляют через var с явным типом или через := с выводом типа. Внутри функции почти всегда используют :=, на уровне пакета — var.",
		Example: `package main

import "fmt"

var globalCount int = 0 // package-level: только var

func main() {
    name := "Alice"        // тип выводится: string
    age := 25              // тип выводится: int
    var height float64 = 1.75 // явный тип, нужен если без инициализации
    var x int              // zero-value: 0
    fmt.Println(name, age, height, x)
}`,
		Concepts: []string{"переменные", "variables", "var", ":=", "zero value", "type inference", "объявление"},
	},
	{
		Name:        "constants",
		DisplayName: "Константы",
		Description: "const объявляет неизменяемое значение времени компиляции. Константы могут быть типизированными или нетипизированными — нетипизированная константа гибче в выражениях (автоматически приводится к нужному типу).",
		Example: `package main

import "fmt"

const Pi = 3.14159           // нетипизированная константа
const MaxRetries int = 3     // типизированная

const (
    StatusOK    = 200
    StatusNotFound = 404
)

func main() {
    fmt.Println(Pi * 2)      // ок: Pi приведётся к float64
    fmt.Println(MaxRetries)
}`,
		Concepts: []string{"константы", "const", "constants", "untyped", "compile-time"},
	},
	{
		Name:        "types",
		DisplayName: "Типы",
		Description: "Go имеет числовые типы (int, int64, float64), строковый, булев, и составные: массивы, слайсы, мапы, структуры, указатели, интерфейсы, функции, каналы. Свой именованный тип создают через type Имя = Базовый или type Имя Базовый (alias vs new type).",
		Example: `package main

import "fmt"

type UserID int64        // новый тип (несовместим с int64 без явного приведения)
type Bytes = []byte      // alias (полностью совместим с []byte)

type Point struct {
    X, Y int
}

func main() {
    var id UserID = 42
    var bs Bytes = []byte{1, 2, 3}
    p := Point{X: 1, Y: 2}
    fmt.Println(id, bs, p)
}`,
		Concepts: []string{"типы", "types", "type", "alias", "named type", "type definition"},
	},

	// ─── Control flow ───────────────────────────────────────────────────
	{
		Name:        "loops",
		DisplayName: "Циклы (for)",
		Description: "В Go ровно один цикл: for. Им же делают while и infinite loop. Перебор коллекций — for ... range.",
		Example: `package main

import "fmt"

func main() {
    // classic for
    for i := 0; i < 3; i++ {
        fmt.Println(i)
    }

    // while-style
    n := 0
    for n < 5 {
        n++
    }

    // infinite + break
    for {
        if n == 5 { break }
        n++
    }

    // range over slice
    nums := []int{10, 20, 30}
    for i, v := range nums {
        fmt.Println(i, v)
    }

    // range over map (порядок не гарантирован!)
    m := map[string]int{"a": 1, "b": 2}
    for k, v := range m {
        fmt.Println(k, v)
    }
}`,
		Concepts: []string{"цикл", "for", "loop", "range", "while", "iterate", "перебор", "итерация"},
	},
	{
		Name:        "if_else",
		DisplayName: "Условия (if/else)",
		Description: "if в Go без скобок вокруг условия, тело обязательно в фигурных. Удобная фишка — короткое объявление прямо в условии: видимость переменной ограничена if-блоком.",
		Example: `package main

import "fmt"

func main() {
    x := 10
    if x > 0 {
        fmt.Println("positive")
    } else if x == 0 {
        fmt.Println("zero")
    } else {
        fmt.Println("negative")
    }

    // объявление в условии — идиоматично для err-чеков
    if val, ok := lookup("foo"); ok {
        fmt.Println("found:", val)
    }
}

func lookup(s string) (string, bool) { return "bar", true }`,
		Concepts: []string{"если", "if", "else", "условие", "branch", "scope"},
	},
	{
		Name:        "switch",
		DisplayName: "switch",
		Description: "switch в Go не проваливается по умолчанию (нет неявного break). Можно делать switch без условия — как цепочку if/else. case может содержать несколько значений через запятую.",
		Example: `package main

import "fmt"

func main() {
    day := "Mon"
    switch day {
    case "Sat", "Sun":
        fmt.Println("weekend")
    case "Mon", "Tue", "Wed", "Thu", "Fri":
        fmt.Println("weekday")
    default:
        fmt.Println("unknown")
    }

    // switch без условия — заменяет лестницу if/else
    n := 42
    switch {
    case n < 0:
        fmt.Println("negative")
    case n == 0:
        fmt.Println("zero")
    case n > 0:
        fmt.Println("positive")
    }
}`,
		Concepts: []string{"switch", "case", "default", "branching", "ветвление"},
	},

	// ─── Composite types ────────────────────────────────────────────────
	{
		Name:        "arrays",
		DisplayName: "Массивы (arrays)",
		Description: "Массив в Go — это значение фиксированной длины. Длина — часть типа: [3]int и [5]int — разные типы. Используются редко, чаще берут slice. Передача массива в функцию копирует его целиком.",
		Example: `package main

import "fmt"

func main() {
    var a [3]int           // [0, 0, 0]
    b := [3]int{1, 2, 3}
    c := [...]int{10, 20, 30, 40} // длину выводит компилятор

    a[0] = 100
    fmt.Println(a, b, c, len(c))
}`,
		Concepts: []string{"массив", "array", "fixed length", "статический массив"},
	},
	{
		Name:        "slices",
		DisplayName: "Слайсы (slices)",
		Description: "Slice — это указатель + длина + capacity на бэкинг-массив. Динамический размер: append возвращает (возможно) новый slice. ВАЖНО: всегда переприсваивай результат append, иначе можно потерять данные.",
		Example: `package main

import "fmt"

func main() {
    s := []int{1, 2, 3}    // slice literal
    s = append(s, 4)       // s = [1 2 3 4]

    // подслайс
    sub := s[1:3]          // [2 3] — разделяет тот же бэкинг!
    sub[0] = 99            // мутирует и s
    fmt.Println(s, sub)    // [1 99 3 4] [99 3]

    // make: preallocate
    buf := make([]int, 0, 100) // len=0, cap=100
    for i := 0; i < 5; i++ {
        buf = append(buf, i)
    }

    // длина и capacity
    fmt.Println(len(buf), cap(buf)) // 5 100
}`,
		Concepts: []string{"слайс", "slice", "append", "len", "cap", "backing array", "подслайс", "make"},
	},
	{
		Name:        "maps",
		DisplayName: "Мапы (map)",
		Description: "map[K]V — хеш-таблица. Перед записью обязательно инициализировать через make или литерал — nil map к записи приведёт к panic. Чтение через двухзначный return: value, ok := m[key].",
		Example: `package main

import "fmt"

func main() {
    // make + Add
    ages := make(map[string]int)
    ages["alice"] = 30
    ages["bob"] = 25

    // литерал
    colors := map[string]string{
        "red":   "#ff0000",
        "green": "#00ff00",
    }

    // безопасное чтение
    if age, ok := ages["alice"]; ok {
        fmt.Println(age)   // 30
    }
    missing, ok := ages["zoe"]
    fmt.Println(missing, ok)  // 0 false

    // удаление
    delete(ages, "alice")

    // итерация (порядок случайный!)
    for k, v := range colors {
        fmt.Println(k, v)
    }
}`,
		Concepts: []string{"мапа", "map", "hash map", "dictionary", "словарь", "delete", "ok-comma idiom"},
	},
	{
		Name:        "structs",
		DisplayName: "Структуры (struct)",
		Description: "struct — именованный набор полей. Объявляется через type Name struct { ... }. Поля с большой буквы экспортируются (видны из других пакетов). Сравниваются по значению если все поля сравнимы.",
		Example: `package main

import "fmt"

type User struct {
    Name string
    Age  int
}

func main() {
    u1 := User{Name: "Alice", Age: 30}  // именованная инициализация
    u2 := User{"Bob", 25}               // позиционная (хрупко при добавлении полей)

    u1.Age++

    fmt.Println(u1.Name, u1.Age)  // Alice 31

    // сравнение
    fmt.Println(u1 == u2)  // false
}`,
		Concepts: []string{"структура", "struct", "field", "поле", "exported", "композитный тип"},
	},
	{
		Name:        "pointers",
		DisplayName: "Указатели",
		Description: "Указатель — значение типа *T, хранит адрес. & берёт адрес переменной, * разыменовывает. Передача указателя в функцию позволяет менять оригинал. Не путать с C: нет арифметики указателей.",
		Example: `package main

import "fmt"

type Counter struct {
    Value int
}

func increment(c *Counter) {  // принимаем указатель
    c.Value++                 // авторазыменование при доступе к полю
}

func main() {
    c := Counter{Value: 0}
    increment(&c)             // передаём адрес
    increment(&c)
    fmt.Println(c.Value)      // 2

    n := 5
    p := &n
    *p = 100                  // разыменование для записи
    fmt.Println(n)            // 100
}`,
		Concepts: []string{"указатель", "pointer", "*", "&", "адрес", "разыменование", "dereference", "address-of"},
	},

	// ─── Functions ──────────────────────────────────────────────────────
	{
		Name:        "functions",
		DisplayName: "Функции",
		Description: "Функция объявляется через func Имя(параметры) тип_возврата { ... }. Может возвращать несколько значений (типичный паттерн: result, err). Тип параметра пишется ПОСЛЕ имени.",
		Example: `package main

import (
    "errors"
    "fmt"
)

// один параметр, один return
func square(x int) int {
    return x * x
}

// два возврата (классический Go-стиль: value, err)
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// named return values
func split(sum int) (x, y int) {
    x = sum * 4 / 9
    y = sum - x
    return
}

func main() {
    fmt.Println(square(5))      // 25

    if result, err := divide(10, 2); err == nil {
        fmt.Println(result)     // 5
    }

    fmt.Println(split(17))      // 7 10
}`,
		Concepts: []string{"функция", "function", "func", "return", "named return", "multiple return"},
	},
	{
		Name:        "variadic",
		DisplayName: "Variadic функции",
		Description: "...T в последнем параметре делает функцию variadic — внутри это slice []T. При вызове можно либо передать значения через запятую, либо распаковать slice через ....",
		Example: `package main

import "fmt"

func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

func main() {
    fmt.Println(sum(1, 2, 3))        // 6
    fmt.Println(sum())               // 0

    nums := []int{4, 5, 6}
    fmt.Println(sum(nums...))        // 15 — распаковка
}`,
		Concepts: []string{"variadic", "переменное число", "args", "rest", "..."},
	},
	{
		Name:        "closures",
		DisplayName: "Замыкания",
		Description: "Функция в Go — first-class value. Можно присваивать переменным, передавать в другие функции, возвращать. Замыкание захватывает переменные окружения по ссылке.",
		Example: `package main

import "fmt"

func makeCounter() func() int {
    count := 0
    return func() int {  // захватывает count
        count++
        return count
    }
}

func main() {
    c := makeCounter()
    fmt.Println(c())   // 1
    fmt.Println(c())   // 2
    fmt.Println(c())   // 3

    // отдельные счётчики не делят состояние
    c2 := makeCounter()
    fmt.Println(c2())  // 1
}`,
		Concepts: []string{"замыкание", "closure", "первоклассная функция", "first-class function", "capture"},
	},

	// ─── Methods, interfaces ────────────────────────────────────────────
	{
		Name:        "methods",
		DisplayName: "Методы",
		Description: "Метод — функция, привязанная к типу через receiver. Receiver-указатель (*T) даёт возможность мутировать; receiver-значение (T) копирует. Идиома: для больших структур или для мутации — указатель.",
		Example: `package main

import "fmt"

type Rectangle struct {
    Width, Height float64
}

// receiver — значение (только чтение)
func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

// receiver — указатель (мутация)
func (r *Rectangle) Scale(factor float64) {
    r.Width *= factor
    r.Height *= factor
}

func main() {
    r := Rectangle{Width: 3, Height: 4}
    fmt.Println(r.Area())   // 12

    r.Scale(2)              // Go автоматически берёт &r
    fmt.Println(r.Area())   // 48
}`,
		Concepts: []string{"метод", "method", "receiver", "value receiver", "pointer receiver"},
	},
	{
		Name:        "interfaces",
		DisplayName: "Интерфейсы",
		Description: "Интерфейс — набор сигнатур методов. Любой тип, реализующий все методы, автоматически удовлетворяет интерфейсу (implicit interfaces, без implements). Идиома: маленькие интерфейсы — io.Reader, io.Writer (по 1 методу).",
		Example: `package main

import "fmt"

type Shape interface {
    Area() float64
}

type Circle struct{ R float64 }
type Square struct{ Side float64 }

func (c Circle) Area() float64 { return 3.14 * c.R * c.R }
func (s Square) Area() float64 { return s.Side * s.Side }

// принимает любой Shape
func describe(s Shape) {
    fmt.Printf("area = %.2f\n", s.Area())
}

func main() {
    describe(Circle{R: 5})     // area = 78.50
    describe(Square{Side: 4})  // area = 16.00

    // type assertion
    var s Shape = Circle{R: 3}
    if c, ok := s.(Circle); ok {
        fmt.Println("radius:", c.R)
    }
}`,
		Concepts: []string{"интерфейс", "interface", "type assertion", "implicit interfaces", "polymorphism", "полиморфизм"},
	},

	// ─── Concurrency ────────────────────────────────────────────────────
	{
		Name:        "goroutines",
		DisplayName: "Горутины",
		Description: "Горутина — лёгкий поток выполнения, запускается через `go funcName()`. Стек начинается с 2-8 KB и растёт по необходимости. Тысячи горутин — норма. Главное правило: НЕ запускай горутину без плана как её ждать или останавливать.",
		Example: `package main

import (
    "fmt"
    "sync"
)

func worker(id int, wg *sync.WaitGroup) {
    defer wg.Done()
    fmt.Printf("worker %d started\n", id)
}

func main() {
    var wg sync.WaitGroup
    for i := 1; i <= 3; i++ {
        wg.Add(1)
        go worker(i, &wg)  // запускаем 3 горутины
    }
    wg.Wait()  // ждём все
    fmt.Println("all done")
}`,
		Concepts: []string{"горутина", "goroutine", "go", "concurrent", "concurrency", "конкурентность", "поток", "thread"},
	},
	{
		Name:        "channels",
		DisplayName: "Каналы (channels)",
		Description: "Канал — типизированная труба для передачи значений между горутинами. Без буфера — отправка блокирует пока кто-то не примет. С буфером — блокирует когда буфер полон. Закрытие через close, чтение из закрытого канала возвращает zero value и ok=false.",
		Example: `package main

import "fmt"

func main() {
    ch := make(chan int)  // unbuffered

    go func() {
        ch <- 42  // отправка
    }()

    val := <-ch   // приём — блокирует пока 42 не придёт
    fmt.Println(val)  // 42

    // buffered channel
    buf := make(chan string, 2)
    buf <- "hello"
    buf <- "world"
    close(buf)

    // range закроется когда канал закрыт
    for msg := range buf {
        fmt.Println(msg)
    }
}`,
		Concepts: []string{"канал", "channel", "chan", "buffered", "unbuffered", "close", "send", "receive", "горутина связь"},
	},
	{
		Name:        "select",
		DisplayName: "select",
		Description: "select — switch для каналов. Ждёт любой из перечисленных case (отправка или приём), выполняет первый готовый. default — неблокирующий вариант. Главный инструмент для таймаутов и отмены.",
		Example: `package main

import (
    "fmt"
    "time"
)

func main() {
    ch := make(chan string)

    go func() {
        time.Sleep(2 * time.Second)
        ch <- "result"
    }()

    select {
    case msg := <-ch:
        fmt.Println("got:", msg)
    case <-time.After(1 * time.Second):
        fmt.Println("timeout")
    }
}`,
		Concepts: []string{"select", "таймаут", "timeout", "wait multiple", "несколько каналов"},
	},
	{
		Name:        "mutex",
		DisplayName: "Мьютекс (sync.Mutex)",
		Description: "sync.Mutex даёт эксклюзивный доступ к ресурсу. Lock/Unlock защищают критическую секцию. Альтернатива каналам когда просто нужно защитить shared state. Идиома: defer mu.Unlock() сразу после Lock.",
		Example: `package main

import (
    "fmt"
    "sync"
)

type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func main() {
    c := &SafeCounter{}
    var wg sync.WaitGroup
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.Inc()
        }()
    }
    wg.Wait()
    fmt.Println(c.count)  // 1000 (без mutex могло быть меньше)
}`,
		Concepts: []string{"мьютекс", "mutex", "lock", "unlock", "race condition", "shared state", "критическая секция"},
	},

	// ─── Errors ─────────────────────────────────────────────────────────
	{
		Name:        "errors",
		DisplayName: "Обработка ошибок",
		Description: "В Go ошибки — обычные значения типа error (интерфейс с методом Error() string). Идиома: возвращай (result, error), проверяй if err != nil. Оборачивай через fmt.Errorf(\"...: %w\", err) для сохранения цепочки.",
		Example: `package main

import (
    "errors"
    "fmt"
    "os"
)

var ErrNotFound = errors.New("not found")  // sentinel error

func loadConfig(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("read %s: %w", path, err)  // wrapping
    }
    return data, nil
}

func main() {
    _, err := loadConfig("missing.yaml")
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            fmt.Println("config missing, using defaults")
        } else {
            fmt.Println("error:", err)
        }
    }
}`,
		Concepts: []string{"ошибка", "error", "errors", "errors.Is", "errors.As", "wrap", "обёртка", "sentinel error"},
	},
	{
		Name:        "defer",
		DisplayName: "defer",
		Description: "defer откладывает вызов функции до момента возврата из enclosing-функции. Выполняется в обратном порядке (LIFO). Главный кейс — гарантированная очистка ресурсов: Close, Unlock, Cancel.",
		Example: `package main

import (
    "fmt"
    "os"
)

func readFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()  // гарантированно закроется

    // читаем... даже если return ниже — f.Close() выполнится
    return nil
}

func main() {
    fmt.Println("start")
    defer fmt.Println("deferred 1")
    defer fmt.Println("deferred 2")
    defer fmt.Println("deferred 3")
    fmt.Println("end")
    // вывод: start → end → deferred 3 → deferred 2 → deferred 1
}`,
		Concepts: []string{"defer", "отложенный вызов", "cleanup", "LIFO", "ресурсы", "Close"},
	},
	{
		Name:        "panic_recover",
		DisplayName: "panic и recover",
		Description: "panic — аварийная остановка горутины с разматыванием стека. recover внутри defer-функции перехватывает panic и возвращает программу к нормальному выполнению. Использовать ТОЛЬКО для непредвиденных ситуаций или на границе HTTP-хендлера.",
		Example: `package main

import "fmt"

func safeDivide(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("recovered: %v", r)
        }
    }()
    return a / b, nil  // паника при b=0
}

func main() {
    r, err := safeDivide(10, 0)
    fmt.Println(r, err)  // 0 recovered: runtime error: integer divide by zero
}`,
		Concepts: []string{"panic", "recover", "паника", "восстановление", "stack unwinding"},
	},

	// ─── Standard library ──────────────────────────────────────────────
	{
		Name:        "fmt_package",
		DisplayName: "Пакет fmt",
		Description: "fmt — форматированный ввод/вывод. Println для отладки, Printf для форматирования, Sprintf если нужна строка. Verbs: %d (int), %s (string), %v (any), %T (тип), %q (escape), %+v (struct с именами полей).",
		Example: `package main

import "fmt"

type User struct {
    Name string
    Age  int
}

func main() {
    u := User{Name: "Alice", Age: 30}

    fmt.Println(u)                  // {Alice 30}
    fmt.Printf("%v\n", u)           // {Alice 30}
    fmt.Printf("%+v\n", u)          // {Name:Alice Age:30}
    fmt.Printf("%#v\n", u)          // main.User{Name:"Alice", Age:30}

    s := fmt.Sprintf("user=%s age=%d", u.Name, u.Age)
    fmt.Println(s)                  // user=Alice age=30

    fmt.Printf("%-10s | %5d\n", "Bob", 25)  // выравнивание
}`,
		Concepts: []string{"fmt", "Printf", "Sprintf", "Println", "форматирование", "format verbs", "%v", "%d", "%s"},
	},
	{
		Name:        "strings_package",
		DisplayName: "Пакет strings",
		Description: "strings — манипуляции со строками. Contains, HasPrefix, HasSuffix, Split, Join, Replace, ToLower, ToUpper, TrimSpace. Строки в Go иммутабельны — каждая операция возвращает новую.",
		Example: `package main

import (
    "fmt"
    "strings"
)

func main() {
    s := "  Hello, World  "
    fmt.Println(strings.TrimSpace(s))           // "Hello, World"
    fmt.Println(strings.ToLower(s))             // "  hello, world  "
    fmt.Println(strings.Contains(s, "World"))   // true
    fmt.Println(strings.Replace(s, "l", "L", -1)) // -1 = all

    csv := "a,b,c,d"
    parts := strings.Split(csv, ",")
    fmt.Println(parts)                          // [a b c d]
    fmt.Println(strings.Join(parts, " | "))     // "a | b | c | d"
}`,
		Concepts: []string{"strings", "Contains", "Split", "Join", "Replace", "TrimSpace", "строки stdlib"},
	},
	{
		Name:        "strconv_package",
		DisplayName: "Пакет strconv",
		Description: "strconv — конвертация строк в числа и обратно. Atoi/Itoa для int, ParseFloat/FormatFloat для float, ParseBool/FormatBool для bool. Все Parse-функции возвращают (value, error).",
		Example: `package main

import (
    "fmt"
    "strconv"
)

func main() {
    n, err := strconv.Atoi("42")
    if err != nil {
        fmt.Println("not a number")
    }
    fmt.Println(n + 1)                     // 43

    s := strconv.Itoa(100)
    fmt.Println(s + "%")                   // "100%"

    f, _ := strconv.ParseFloat("3.14", 64)
    fmt.Println(f * 2)                     // 6.28

    fs := strconv.FormatFloat(3.14159, 'f', 2, 64)
    fmt.Println(fs)                        // "3.14"
}`,
		Concepts: []string{"strconv", "Atoi", "Itoa", "ParseFloat", "ParseInt", "конвертация", "string to int"},
	},

	// ─── Idioms ─────────────────────────────────────────────────────────
	{
		Name:        "ok_idiom",
		DisplayName: "ok-comma идиома",
		Description: "Особый return из map[k]/type assertion/<-channel возвращает второе значение bool. Игнорировать его опасно — отличает 'значения нет' от 'значение есть и оно zero'.",
		Example: `package main

import "fmt"

func main() {
    m := map[string]int{"a": 0}

    v1 := m["a"]      // 0
    v2 := m["missing"] // 0 — но это zero value, нет различия!
    fmt.Println(v1, v2)

    // Правильно:
    if v, ok := m["a"]; ok {
        fmt.Println("a:", v)  // a: 0 (есть)
    }
    if _, ok := m["missing"]; !ok {
        fmt.Println("missing!")  // missing!
    }

    // То же с type assertion
    var x interface{} = "hello"
    if s, ok := x.(string); ok {
        fmt.Println("string:", s)
    }
}`,
		Concepts: []string{"ok-comma", "two-value return", "exists check", "проверка существования"},
	},
	{
		Name:        "zero_values",
		DisplayName: "Zero values",
		Description: "Любая переменная без явной инициализации получает zero value своего типа: int → 0, string → \"\", bool → false, pointer/slice/map/channel/func/interface → nil. Это позволяет писать код без явной инициализации в простых случаях.",
		Example: `package main

import "fmt"

type Server struct {
    Name string
    Port int
    TLS  bool
}

func main() {
    var n int          // 0
    var s string       // ""
    var b bool         // false
    var p *int         // nil
    var sl []int       // nil (но len=0, можно append)
    var m map[string]int // nil — НЕЛЬЗЯ писать без make!

    fmt.Println(n, s, b, p, sl, m == nil)

    srv := Server{}  // все поля zero: "", 0, false
    fmt.Printf("%+v\n", srv)  // {Name: Port:0 TLS:false}
}`,
		Concepts: []string{"zero value", "нулевое значение", "default value", "nil", "init"},
	},

	// ─── Build / modules ────────────────────────────────────────────────
	{
		Name:        "modules",
		DisplayName: "Модули (go.mod)",
		Description: "Go-модуль — это директория с go.mod в корне. go mod init <name> создаёт его. go get <package> добавляет зависимость. go mod tidy чистит лишние. Импорт от корня модуля — github.com/user/repo/pkg.",
		Example: `# Initialize
go mod init github.com/myuser/myapp

# Add dependency
go get github.com/google/uuid

# Resulting go.mod
module github.com/myuser/myapp

go 1.22

require github.com/google/uuid v1.6.0

# Clean unused
go mod tidy

# Build / run
go run .
go build ./...
go test ./...`,
		Concepts: []string{"модули", "modules", "go mod", "go.mod", "go get", "зависимости", "dependencies"},
	},
	{
		Name:        "testing",
		DisplayName: "Тестирование",
		Description: "Тесты в Go — файлы с суффиксом _test.go в том же пакете. Функция func TestXxx(t *testing.T). Запуск через go test. Подтесты через t.Run для table-driven тестов. testify не нужен — стандартная либа покрывает 90%.",
		Example: `package mymath

import "testing"

func Add(a, b int) int { return a + b }

func TestAdd(t *testing.T) {
    cases := []struct {
        name     string
        a, b     int
        want     int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := Add(tc.a, tc.b); got != tc.want {
                t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.want)
            }
        })
    }
}

// запуск: go test ./...
// запуск с покрытием: go test -cover ./...`,
		Concepts: []string{"тесты", "testing", "test", "table-driven", "subtests", "go test", "coverage"},
	},
}
