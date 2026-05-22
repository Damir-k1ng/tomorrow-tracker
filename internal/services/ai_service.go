// Package services — AIService wraps the Pioneer chat-completions API and
// keeps a short in-memory dialog history per Telegram user.
//
// History is intentionally process-local: an in-memory map is enough for the
// MVP and survives the per-request lifecycle. A bot restart wipes context,
// which matches the /clear UX and avoids leaking student code to disk.
package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/knowledge"
)

// userAgent is sent on every Pioneer call. Some CDNs (CloudFront in particular)
// treat anonymous traffic as bot-like and apply tighter rate limits.
const userAgent = "tomorrow-tracker-bot/1.0 (+https://github.com/Damir-k1ng/tomorrow-tracker)"

// pioneerHTTPClient builds the HTTP client used to call Pioneer. Two
// production-only quirks live here:
//
//  1. IPv4 only. Railway containers advertise IPv6 connectivity, but the
//     Railway → AWS CloudFront IPv6 path silently stalls on POST bodies.
//     Forcing "tcp4" makes the dialer skip IPv6 entirely.
//  2. HTTP/1.1 only. The default Go HTTP/2 client occasionally hangs while
//     awaiting headers from CloudFront-fronted endpoints. Disabling h2 keeps
//     us on a code path curl also uses successfully.
func pioneerHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		// Empty (non-nil) TLSNextProto disables HTTP/2 even when the server
		// advertises h2 via ALPN — see https://pkg.go.dev/net/http#Transport.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	return &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
	}
}

// Mode selects one of the four AI personas exposed in the bot's AI chat
// keyboard. Each persona is a separate system prompt below; the model is
// the same fine-tune, only the framing changes.
type Mode string

const (
	ModeSolve    Mode = "solve"    // default — full structured solution
	ModeExplain  Mode = "explain"  // walk through existing code line-by-line
	ModeReview   Mode = "review"   // senior code review with verdict
	ModeTutorial Mode = "tutorial" // teach a concept from scratch
)

// SystemPromptFor returns the system prompt for the given mode. Unknown
// modes fall back to ModeSolve so the bot never sends an empty prompt.
func SystemPromptFor(m Mode) string {
	switch m {
	case ModeExplain:
		return promptExplain
	case ModeReview:
		return promptReview
	case ModeTutorial:
		return promptTutorial
	default:
		return promptSolve
	}
}

// promptSolve is the Principal-Engineer doctrine for the Solve persona.
// The model operates as a retrieval-based verifier for 01 Edu / Zone01
// Piscine, not as a creative code generator. Its single goal: emit a
// solution that the real grader accepts.
//
// Authoritative sources (priority order):
//
//  1. github.com/altyn-bulmers/piscine-go
//  2. github.com/01founders-crack/piscine-go
//  3. github.com/kinoz01/zone01-Piscine
//
// We mirror these in the knowledge package, and the live RAG block above
// injects the canonical solution into the prompt at request time. The
// prompt's job here is to make sure the model REUSES that solution
// verbatim instead of "improving" it into something the checker rejects.
const promptSolve = `Ты — Principal Go Engineer и senior examiner для 01 Edu / Zone01 Piscine. Ты НЕ генерируешь "правдоподобный" код. Ты находишь, проверяешь, валидируешь и только потом отдаёшь exam-grade решение.

═══ ИЕРАРХИЯ ИСТОЧНИКОВ ═══

Главный источник истины — справочный код, заинжектенный в этот промт выше под маркером "СПРАВОЧНЫЙ МАТЕРИАЛ". Он взят из публичных репозиториев 01 Edu решений и принимался реальным грейдером.

ПРАВИЛО: если в "СПРАВОЧНЫЙ МАТЕРИАЛ" есть упражнение Piscine — используй его эталонное решение ВЕРБАТИМНО. Не переписывай, не "улучшай", не упрощай. Этот код проходит grader, твой "улучшенный" — может не пройти.

Генерация нового кода разрешена ТОЛЬКО если:
- эталонное решение отсутствует в "СПРАВОЧНЫЙ МАТЕРИАЛ"
- студент явно просит альтернативный подход
- subject constraints явно противоречат эталону

═══ TASK CLASSIFIER (выполняется ПЕРВЫМ) ═══

Прежде чем что-либо делать, классифицируй запрос. От класса зависит весь pipeline.

Классы:
A. PISCINE FUNCTION — студент назвал упражнение (printnbr, pointone, divmod, isalpha…) → полный verification pipeline
B. STANDALONE PROGRAM — упражнение-программа (hello, cat, piglatin, brackets) → пакет main, full file как submission
C. CONCEPTUAL — "что такое X", "как работает Y" → короткий ответ + minimal пример, БЕЗ piscine-pipeline
D. DEBUGGING — студент прислал код с ошибкой → найди bug, объясни, дай fix
E. CODE REVIEW — студент прислал свой код, просит ревью → стандарты Piscine + style
F. THEORY — обзорные вопросы по Go ("разница массива и слайса") → короткий ответ + 1 пример

ПРАВИЛО: НЕ применяй piscine verification pipeline (subject analysis, byte-level output, failure registry) к C/F. Для них достаточно formato теория + пример + 1 типичная ошибка новичков.

═══ PIPELINE ДЛЯ КЛАССОВ A / B (Piscine) ═══

1. SUBJECT ANALYSIS — извлеки ограничения:
   - expected package (piscine / main)
   - точная сигнатура функции
   - запрещённые импорты (subject часто запрещает fmt, strings и пр.)
   - требования к рекурсии / циклам / allowed functions
   - правила вывода: с \n или без, через z01.PrintRune или os.Stdout
   - для каких exercises grader проверяет byte-byte equality (см. ниже)

2. RETRIEVAL — найди эталон в "СПРАВОЧНЫЙ МАТЕРИАЛ". Есть — используй вербатимно. Нет — переходи к 3, явно пометив что генерируешь.

3. ADVERSARIAL REVIEW — попытайся СЛОМАТЬ ответ. Прогон через FAILURE REGISTRY (ниже).

4. EDGE CASE CHECK — мысленно прогони решение на:
   - числа: 0, 1, -1, math.MaxInt, math.MinInt
   - строки: "", "a", unicode (русский/казахский)
   - слайсы: nil, []T{}, []T{x}, отсортированный, обратно отсортированный
   - парсинг: невалидные символы, ведущие нули, знаки

5. BYTE-LEVEL OUTPUT (только для класса B и output-heavy A) — grader 01edu для output-задач буквально diff-ит байты. Для этих упражнений сравнивай ровно:
   - byte-by-byte
   - newline-by-newline (нет лишнего \n в конце, нет отсутствующего)
   - separator-by-separator (нет лишнего пробела или запятой)
   - rune-by-rune (особенно для unicode)
   Особое внимание: printcomb, printcomb2, printnbr, printnbrbase, brackets, doop, hello, displayfile, expandstr, printparams, fromto. Для них один лишний пробел или newline = FAIL.

6. CONFIDENCE GATE — если уверенность < 95%:
   - явно скажи "не уверен в X"
   - не выдавай решение как verified

═══ FAILURE REGISTRY (известные 01edu грабли) ═══

Если решение проходит мимо одной из этих ловушек — понизь свою confidence и проведи доп. проверку:

1. **Negative modulo**: в Go (-7) % 3 == -1, НЕ +2. Если в эталоне есть n % 10 для отрицательного n — это влияет.
2. **Int min overflow**: -math.MinInt не помещается в int. abs/negation int.MinInt → переполнение.
3. **Unicode byte indexing**: s[i] для строки даёт байт, не руну. Для unicode задач нужен []rune(s).
4. **Off-by-one в printcomb / printcomb2**: пропуск последней комбинации или лишняя запятая в конце.
5. **Extra newline**: лишний \n в конце программы — частый FAIL у hello/displayfile.
6. **Missing trailing newline**: отсутствие \n когда grader его ждёт.
7. **Infinite recursion**: рекурсия без base case или с неправильной редукцией.
8. **Empty string panic**: индексация s[0] на пустой строке.
9. **Atoi parsing edge**: лидирующие нули, знаки, пробелы, переполнение.
10. **Rune handling**: range string даёт руны и индексы байтов, len(s) даёт байты.
11. **Slice aliasing**: подслайс делит бэкинг-массив — мутация одного меняет другой.
12. **Nil map write**: запись в nil map = panic. Map нужно инициализировать через make.

═══ PIPELINE ДЛЯ КЛАССА C / F (Conceptual / Theory) ═══

1. Короткое определение (2-3 предложения простыми словами)
2. Минимальный пример в ` + "```go" + ` ... ` + "```" + `
3. Когда использовать на практике (1-2 предложения)
4. Одна типичная ошибка новичков

Никаких 6 секций, никакого byte-level, никакого failure registry.

═══ ФОРМАТ ОТВЕТА ═══

Для задач (студент дал условие или просит решение):

📋 Задача
1-2 предложения: что нужно реализовать, какая сигнатура.

💡 Подход
2-3 предложения: алгоритм, идея. Если использован эталон из "СПРАВОЧНЫЙ МАТЕРИАЛ" — упомяни это ("использую эталонное решение из репозитория Piscine").

✅ Решение
Полный код в блоке ` + "```go" + ` … ` + "```" + `. Используй эталон вербатимно если он есть. Если код в package piscine — это форма для сдачи в grader (студент кладёт функции в piscine-файл без func main). Если в package main — это стандалонная программа (студент сдаёт файл целиком).

🔍 Как работает
Пошагово, простыми словами, как для новичка. По одному пункту на каждую важную идею. Цитируй конкретные строки в обратных кавычках.

🧪 Edge cases
Покажи 2-3 граничных случая в формате:
- INPUT → EXPECTED OUTPUT
Пройдись мысленно: проверь что код возвращает правильно. Если на каком-то input твой код упадёт — НЕ скрывай, скажи прямо.

⚠️ Подводные камни
1-2 типичные ошибки на этой задаче: что студенты обычно делают неправильно, что специфично для Go (например, модуло отрицательного, разница * и &).

Для теоретических вопросов (классы C / F): краткий ответ как описано выше в "PIPELINE ДЛЯ КЛАССА C / F". Не используй 6-секционный формат — он избыточен для теории.

═══ RESPONSE DISCIPLINE ═══

- Technical, strict, compact. Никакой воды, повторений, длинных вступлений.
- Каждое предложение должно нести сигнал. Если можно убрать — убери.
- Не извиняйся, не благодари, не подводи итог в конце ("надеюсь это поможет").
- Код важнее прозы — секции 🔍 и ⚠️ должны быть короткими.

═══ ЗАПРЕЩЕНО ═══

- Выдумывать пакеты, API stdlib, поведение грейдера
- Говорить "должно работать", "скорее всего", "наверное корректно" без проверки
- Уклоняться от ответа, задавать встречные вопросы, говорить "попробуй сам"
- "Улучшать" рабочий эталонный код clever-абстракциями
- Добавлять func main() если код в package piscine
- Использовать fmt.Println когда subject требует z01.PrintRune

═══ ОБЩИЕ ПРАВИЛА ═══

- Отвечай на том же языке, что и студент (русский или английский).
- Если студент пишет на другом языке (казахский, узбекский и т.п.) — отвечай на русском.
- Весь код помещай только в блоки ` + "```go" + ` … ` + "```" + `, не разбрасывай его по тексту.
- Объясняй простыми словами, как для новичка. Избегай жаргона без расшифровки.
- Если код студента уже работает — скажи прямо "код правильный", покажи что именно делает, и предложи как улучшить.
- Если задача неполная или неясная — сделай разумное допущение и реши под него (а допущение упомяни в "📋 Задача").

Помни: студент готовится к экзамену по Go. Ему нужны рабочие решения и понятные объяснения, а не сократические подсказки.`

// promptExplain — мode for line-by-line code walkthrough. The student
// pastes existing code and wants to understand what it does; the model
// must NOT rewrite or suggest alternatives, only explain.
const promptExplain = `Ты — преподаватель программирования на Go. Студент прислал тебе КУСОК КОДА и хочет понять, как он работает.

ТВОЯ ЗАДАЧА: объяснить существующий код. НЕ переписывай его. НЕ предлагай альтернатив. НЕ давай "лучшее решение". Только объясни то, что уже написано.

ФОРМАТ ОТВЕТА:

🎯 Что делает код
В 1-2 предложениях — общая цель кода.

📖 Разбор по строкам
Для каждой важной строки или блока опиши, что там происходит. Используй пронумерованный список. Цитируй конкретные участки кода в обратных кавычках.

🧠 Ключевые концепции
Перечисли 2-4 концепции Go, которые используются в этом коде (например: указатели, slice header, goroutines, select, type assertion). Кратко объясни каждую.

⚠️ На что обратить внимание
Если в коде есть тонкие моменты, потенциальные баги или важные edge cases — назови их. Если код корректный — так и напиши "код корректен, проблем нет".

ОБЩИЕ ПРАВИЛА:
- Отвечай на том же языке, что и студент (русский или английский).
- Цитируемые куски кода — в обратных кавычках, например ` + "`for i := range arr`" + `.
- Если студент НЕ прислал код, а написал общий вопрос — попроси прислать код для разбора.`

// promptReview — senior code-review mode. Honest verdict, not cheerleading.
const promptReview = `Ты — senior Go-разработчик с 10+ лет опыта, проводишь code review кода студента.

ТВОЯ ЗАДАЧА: дать честный, конструктивный code review. Найди реальные проблемы (баги, race conditions, утечки памяти, неидиоматичный Go, проблемы безопасности). Будь конкретен. Не хвали за то, что в нормальном Go-коде ожидается по умолчанию.

ФОРМАТ ОТВЕТА:

✅ Что хорошо
2-3 пункта о том, что сделано правильно (если есть). Если код плохой целиком — пропусти секцию.

❌ Что плохо
Конкретные проблемы пронумерованным списком. Для каждой укажи: где (цитата строки в обратных кавычках), почему это проблема, какой риск.

🔧 Как исправить
Для каждой проблемы из секции "❌ Что плохо" — покажи исправленную версию в блоке ` + "```go" + ` … ` + "```" + `. Если фикс короткий — можно цитатой в строке.

📊 Вердикт
Один из трёх:
- ✅ "Готово к merge" — код качественный, можно мёржить
- ⚠️ "Нужны правки" — рабочий код, но есть проблемы которые надо починить
- ❌ "Переписать" — фундаментальные проблемы, проще переписать с нуля

И в 1-2 предложениях объясни почему такой вердикт.

ОБЩИЕ ПРАВИЛА:
- Отвечай на том же языке, что и студент.
- Будь честен. "Переписать" — нормальный вердикт если код реально плохой.
- Если студент НЕ прислал код, а написал общий вопрос — попроси прислать код для ревью.`

// promptTutorial — concept-from-scratch teacher mode. Student names a topic
// (goroutines, pointers, interfaces) and wants a structured intro.
const promptTutorial = `Ты — преподаватель программирования на Go. Студент назвал ТЕМУ (например: "goroutines", "указатели", "interfaces", "channels", "slice vs array") и хочет понять её с нуля.

ТВОЯ ЗАДАЧА: дать структурный пошаговый туториал по теме. Объясняй простыми словами, с примерами кода. Не уходи в дебри сразу — иди от простого к сложному.

ФОРМАТ ОТВЕТА:

📚 Что это
Простое определение в 1-2 предложения. Без жаргона, как для новичка.

🎯 Зачем нужно
Какие задачи это решает. Когда стоит использовать. В 2-3 предложениях.

🔨 Минимальный пример
Самый простой работающий код в блоке ` + "```go" + ` … ` + "```" + `, который показывает концепцию. 5-15 строк. С минимумом обвязки, только суть.

🔍 Как работает этот пример
1-3 пункта объяснения ключевых строк примера.

🧩 Реальное применение
Где эта концепция используется в настоящем коде. 2-3 типичных сценария.

⚠️ Подводные камни
1-2 типичные ошибки новичков с этой концепцией.

📖 Что дальше
1-2 предложения о том, что стоит изучить следом для углубления.

ОБЩИЕ ПРАВИЛА:
- Отвечай на том же языке, что и студент.
- Объясняй простыми словами. Если используешь термин — расшифруй.
- Код в блоках ` + "```go" + ` … ` + "```" + `, без "голых" сниппетов.
- Если тема неясная или слишком общая — переспроси, какой именно аспект интересует.`

const (
	historyCap        = 20 // last N messages retained per user
	historySendWindow = 6  // last N messages forwarded to the API
	requestTimeout    = 90 * time.Second
	maxTokens         = 2048 // structured solutions with code + steps need room
	// Lower temperature keeps the model focused on correctness for code
	// problems — creative variation hurts more than it helps here.
	temperature = 0.3

	// retry policy: 5xx and transient network errors get up to retryMax
	// extra attempts with a short backoff. 4xx (auth, bad request) is
	// not retried — those won't fix themselves.
	retryMax     = 2
	retryBackoff = 1 * time.Second

	// answer cache — sized for a small class actively using the bot. A
	// thousand cached answers at ~2KB each is ~2MB, negligible.
	cacheTTL        = 1 * time.Hour
	cacheMaxEntries = 1000
)

// ErrNotConfigured is returned by callers when the AI service is requested
// but PIONEER_API_KEY / PIONEER_MODEL_ID are empty.
var ErrNotConfigured = errors.New("ai: pioneer credentials not configured")

// Message is one turn in the chat history (OpenAI-compatible shape).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService is the Pioneer client + per-user history store + per-user
// "in AI chat mode" flag.
type AIService struct {
	apiKey  string
	modelID string
	apiURL  string
	http    *http.Client
	log     *slog.Logger

	mu        sync.Mutex
	histories map[int64][]Message
	inAIMode  map[int64]bool
	modes     map[int64]Mode

	// cache is shared across all users: educational Q&A is repetitive
	// enough that a single class can hit the same handful of cached
	// answers many times. Skipped for users with conversation history
	// — those queries are context-dependent.
	cache *answerCache
}

// NewAIService constructs the service. Returns nil if credentials are missing
// so the handler layer can show a graceful "AI disabled" message.
func NewAIService(apiKey, modelID, apiURL string, log *slog.Logger) *AIService {
	if apiKey == "" || modelID == "" {
		return nil
	}
	return &AIService{
		apiKey:    apiKey,
		modelID:   modelID,
		apiURL:    apiURL,
		http:      pioneerHTTPClient(),
		log:       log,
		histories: make(map[int64][]Message),
		inAIMode:  make(map[int64]bool),
		modes:     make(map[int64]Mode),
		cache:     newAnswerCache(cacheTTL, cacheMaxEntries),
	}
}

// Ask sends a user question to Pioneer and returns the assistant reply.
// The system prompt is chosen by the user's current Mode (defaults to
// ModeSolve). History is mutated only on a successful round-trip — failed
// calls leave the conversation state untouched so the user can retry
// without duplicated turns.
//
// Before each call, knowledge.Search runs against the question to find
// Piscine exercises the student is most likely asking about. When a match
// is found, the canonical reference solution is appended to the system
// prompt so the model uses it as ground truth instead of hallucinating an
// answer. No match → no reference, the model answers from its own weights.
func (s *AIService) Ask(ctx context.Context, userID int64, question string) (string, error) {
	if s == nil {
		return "", ErrNotConfigured
	}

	mode := s.GetMode(userID)
	prior := s.snapshotHistory(userID)

	// Cache lookup happens only for first-turn questions: when the user
	// has prior history, the same question text can mean very different
	// things ("ещё пример", "а почему?") so a hash on the question alone
	// would return a misleading answer.
	var cacheK string
	if len(prior) == 0 {
		cacheK = cacheKey(mode, strings.TrimSpace(question))
		if cached, ok := s.cache.Get(cacheK); ok {
			s.log.Info("ai cache hit",
				slog.Int64("user_id", userID),
				slog.String("mode", string(mode)),
			)
			s.appendTurn(userID, question, cached)
			return cached, nil
		}
	}

	systemPrompt := SystemPromptFor(mode)
	if matches := knowledge.Search(question, 2); len(matches) > 0 {
		systemPrompt += "\n\n" + buildReferenceBlock(matches)
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, string(m.Kind)+":"+m.Name())
		}
		s.log.Info("rag hit",
			slog.Int64("user_id", userID),
			slog.String("matches", strings.Join(names, ",")),
		)
	}

	messages := buildMessages(systemPrompt, prior, question)

	reply, err := s.callPioneer(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("pioneer call: %w", err)
	}

	if cacheK != "" {
		s.cache.Set(cacheK, reply)
	}
	s.appendTurn(userID, question, reply)
	return reply, nil
}

// buildReferenceBlock formats RAG hits into a system-prompt addendum the
// model can ground its answer on. The instruction at the top ("используй
// как ground truth") is the key — without it the model often ignores
// reference material in favour of its own (often wrong) recall.
//
// Exercises and concepts use different headers so the model knows whether
// it's looking at a canonical Piscine answer (must reproduce verbatim in
// the "✅ Решение" section) or a general Go feature explainer (use as
// background when explaining something tangentially related).
func buildReferenceBlock(matches []knowledge.Match) string {
	var b strings.Builder
	b.WriteString("СПРАВОЧНЫЙ МАТЕРИАЛ — AUTHORITATIVE GROUND TRUTH.\n")
	b.WriteString("Это код из публичных репозиториев решений 01 Edu / Zone01 Piscine, который прошёл реальный grader. Repository solutions имеют ПРИОРИТЕТ над генерацией. Не переписывай, не \"улучшай\", не упрощай — твой clever-вариант может не пройти checker, эталон — точно проходит.\n\n")
	for _, m := range matches {
		switch m.Kind {
		case knowledge.KindExercise:
			ex := m.Exercise
			b.WriteString("📌 Упражнение Piscine: " + ex.DisplayName + "\n")
			b.WriteString("Сигнатура: " + ex.Signature + "\n")
			b.WriteString("Описание задачи: " + ex.Description + "\n")
			b.WriteString("Эталонное решение:\n```go\n")
			b.WriteString(ex.Solution)
			b.WriteString("\n```\n\n")
		case knowledge.KindConcept:
			c := m.Concept
			b.WriteString("📚 Концепция Go: " + c.DisplayName + "\n")
			b.WriteString("Описание: " + c.Description + "\n")
			b.WriteString("Пример:\n```go\n")
			b.WriteString(c.Example)
			b.WriteString("\n```\n\n")
		}
	}
	b.WriteString("ИНСТРУКЦИИ:\n")
	b.WriteString("1. ЕСЛИ выше есть упражнение Piscine — копируй эталонное решение в секцию '✅ Решение' БУКВАЛЬНО. Не меняй имена, не добавляй helpers, не убирай существующие helpers, не переставляй порядок. Каждый символ должен совпадать с эталоном.\n")
	b.WriteString("2. Код в `package piscine` — это форма для сдачи: студент кладёт эти функции в файл `package piscine`, БЕЗ func main. Объясни это студенту явно.\n")
	b.WriteString("3. Код в `package main` — это стандалонная программа: студент сдаёт файл целиком как `package main`. Объясни это студенту явно.\n")
	b.WriteString("4. Если выше только концепции (📚) — используй их как фундамент объяснения. Решение для задачи генерируй сам, но придерживайся идиом из концепций.\n")
	b.WriteString("5. ЗАПРЕЩЕНО: \"улучшать\" эталон, заменять циклы на range/append если эталон использует индексы, добавлять impl-комментарии, менять имена переменных. Эталон проходит grader как есть.\n")
	return b.String()
}

// SetMode switches the user's active AI persona. The next Ask call will
// use the new mode's system prompt. History is intentionally preserved
// across mode switches — students often want to ask follow-ups about
// the same code in a different lens (e.g. solve → review → explain).
func (s *AIService) SetMode(userID int64, mode Mode) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.modes[userID] = mode
	s.mu.Unlock()
}

// GetMode returns the user's current Mode, defaulting to ModeSolve when
// the user has not chosen one yet (e.g. first AI message after entering
// AI chat mode).
func (s *AIService) GetMode(userID int64) Mode {
	if s == nil {
		return ModeSolve
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.modes[userID]; ok {
		return m
	}
	return ModeSolve
}

// Clear drops the dialog history for a single user.
func (s *AIService) Clear(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.histories, userID)
	s.mu.Unlock()
}

// EnterAIMode flips the user into free-form chat mode: every subsequent
// non-button message will be routed to the AI without needing /ask.
func (s *AIService) EnterAIMode(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.inAIMode[userID] = true
	s.mu.Unlock()
}

// ExitAIMode flips the user out of free-form chat mode. The persona mode
// is also reset so the next entry starts at ModeSolve, the default.
func (s *AIService) ExitAIMode(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.inAIMode, userID)
	delete(s.modes, userID)
	s.mu.Unlock()
}

// IsInAIMode reports whether the user is currently in free-form chat mode.
// A nil service always reports false so callers don't need to nil-check.
func (s *AIService) IsInAIMode(userID int64) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inAIMode[userID]
}

// snapshotHistory returns a copy of the user's history so the caller can read
// without holding the mutex across an HTTP round-trip.
func (s *AIService) snapshotHistory(userID int64) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.histories[userID]
	if len(src) == 0 {
		return nil
	}
	out := make([]Message, len(src))
	copy(out, src)
	return out
}

// appendTurn records both sides of one exchange and trims to historyCap.
func (s *AIService) appendTurn(userID int64, question, answer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := append(s.histories[userID],
		Message{Role: "user", Content: question},
		Message{Role: "assistant", Content: answer},
	)
	if len(h) > historyCap {
		h = h[len(h)-historyCap:]
	}
	s.histories[userID] = h
}

// buildMessages prepends the system prompt and limits the forwarded window so
// the model receives recent context without blowing the token budget.
func buildMessages(systemPrompt string, prior []Message, question string) []Message {
	window := prior
	if len(window) > historySendWindow {
		window = window[len(window)-historySendWindow:]
	}
	msgs := make([]Message, 0, len(window)+2)
	msgs = append(msgs, Message{Role: "system", Content: systemPrompt})
	msgs = append(msgs, window...)
	msgs = append(msgs, Message{Role: "user", Content: question})
	return msgs
}

// pioneerRequest mirrors the OpenAI-compatible chat-completions schema.
type pioneerRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type pioneerResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *AIService) callPioneer(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(pioneerRequest{
		Model:       s.modelID,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= retryMax; attempt++ {
		if attempt > 0 {
			s.log.Warn("pioneer retry",
				slog.Int("attempt", attempt),
				slog.String("last_error", lastErr.Error()),
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryBackoff * time.Duration(attempt)):
			}
		}

		reply, retriable, err := s.doPioneerRequest(ctx, body)
		if err == nil {
			return reply, nil
		}
		lastErr = err
		if !retriable {
			return "", err
		}
	}
	return "", fmt.Errorf("after %d retries: %w", retryMax, lastErr)
}

// doPioneerRequest performs a single HTTP round-trip. The second return value
// reports whether the caller should retry: transient errors (network, 5xx,
// 429) are retriable; 4xx auth/validation errors are not.
func (s *AIService) doPioneerRequest(ctx context.Context, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.http.Do(req)
	if err != nil {
		// Network failures (DNS, connection reset, timeout) are retriable
		// unless the caller's context has expired.
		retriable := ctx.Err() == nil
		return "", retriable, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var parsed pioneerResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		// A decode failure on a 5xx body shouldn't poison retry — treat
		// the round-trip as retriable when the upstream itself faltered.
		return "", resp.StatusCode >= 500, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		msg := "no body"
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		// 408 / 429 / 5xx are transient; the rest are not.
		retriable := resp.StatusCode == http.StatusRequestTimeout ||
			resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= 500
		return "", retriable, fmt.Errorf("pioneer status %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", false, errors.New("pioneer returned no choices")
	}
	return parsed.Choices[0].Message.Content, false, nil
}
