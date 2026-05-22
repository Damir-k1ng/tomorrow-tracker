package knowledge

// Exercises is the curated set of Piscine Go reference solutions. Each entry
// is the canonical answer the grader at 01.tomorrow-school.ai accepts plus
// the metadata that lets Search surface it from a Russian student question.
//
// Provenance: signatures and solutions are derived from a public Piscine
// solutions repository (gitea.com/mohammed_ramdani/allprojects). Empty
// skeletons in that source are replaced here with working reference
// implementations; non-trivial solutions are kept as-is.
var Exercises = []Exercise{
	// ─── Basics: I/O, simple arithmetic ─────────────────────────────────
	{
		Name:        "pointone",
		DisplayName: "PointOne",
		Signature:   "func PointOne(n *int)",
		Solution: `package piscine

func PointOne(n *int) {
	*n = 1
}`,
		Description: "Принимает указатель на int и записывает по нему значение 1.",
		Concepts:    []string{"указатель", "pointer", "разыменование", "dereference", "*int"},
	},
	{
		Name:        "ultimatepointone",
		DisplayName: "UltimatePointOne",
		Signature:   "func UltimatePointOne(n ***int)",
		Solution: `package piscine

func UltimatePointOne(n ***int) {
	***n = 1
}`,
		Description: "Принимает тройной указатель на int и записывает значение 1 через тройное разыменование.",
		Concepts:    []string{"тройной указатель", "***int", "triple pointer", "указатель на указатель"},
	},
	{
		Name:        "swap",
		DisplayName: "Swap",
		Signature:   "func Swap(a, b *int)",
		Solution: `package piscine

func Swap(a, b *int) {
	*a, *b = *b, *a
}`,
		Description: "Меняет местами значения двух int через указатели.",
		Concepts:    []string{"swap", "обмен", "указатели", "multiple assignment", "*int"},
	},
	{
		Name:        "divmod",
		DisplayName: "DivMod",
		Signature:   "func DivMod(a int, b int, div *int, mod *int)",
		Solution: `package piscine

func DivMod(a int, b int, div *int, mod *int) {
	*div = a / b
	*mod = a % b
}`,
		Description: "Делит a на b, записывает частное в *div и остаток в *mod.",
		Concepts:    []string{"деление", "остаток", "модуль", "div", "mod", "modulo"},
	},
	{
		Name:        "ultimatedivmod",
		DisplayName: "UltimateDivMod",
		Signature:   "func UltimateDivMod(a *int, b *int)",
		Solution: `package piscine

func UltimateDivMod(a *int, b *int) {
	div := *a / *b
	mod := *a % *b
	*a = div
	*b = mod
}`,
		Description: "Сохраняет в *a частное a/b, в *b — остаток a%b. Использует указатели и для входа, и для выхода.",
		Concepts:    []string{"деление с остатком", "in-place", "указатели вход и выход"},
	},
	{
		Name:        "isnegative",
		DisplayName: "IsNegative",
		Signature:   "func IsNegative(nb int)",
		Solution: `package piscine

import "github.com/01-edu/z01"

func IsNegative(nb int) {
	if nb < 0 {
		z01.PrintRune('T')
	} else {
		z01.PrintRune('F')
	}
}`,
		Description: "Печатает 'T' если число отрицательное, иначе 'F'. Использует z01.PrintRune.",
		Concepts:    []string{"отрицательное", "negative", "z01.PrintRune", "сравнение", "if"},
	},
	{
		Name:        "fibonacci",
		DisplayName: "Fibonacci",
		Signature:   "func Fibonacci(index int) int",
		Solution: `package piscine

func Fibonacci(index int) int {
	if index < 0 {
		return -1
	}
	if index == 0 {
		return 0
	}
	if index == 1 {
		return 1
	}
	return Fibonacci(index-1) + Fibonacci(index-2)
}`,
		Description: "Возвращает n-е число Фибоначчи рекурсивно. Для отрицательных индексов возвращает -1.",
		Concepts:    []string{"фибоначчи", "fibonacci", "рекурсия", "recursion", "базовый случай"},
	},
	{
		Name:        "iterativefactorial",
		DisplayName: "IterativeFactorial",
		Signature:   "func IterativeFactorial(nb int) int",
		Solution: `package piscine

func IterativeFactorial(nb int) int {
	if nb < 0 || nb > 20 {
		return 0
	}
	result := 1
	for i := 2; i <= nb; i++ {
		result *= i
	}
	return result
}`,
		Description: "Считает факториал n итеративно (циклом). Возвращает 0 для отрицательных и значений больше 20 (переполнение).",
		Concepts:    []string{"факториал", "factorial", "цикл", "for", "переполнение", "overflow"},
	},
	{
		Name:        "iterativepower",
		DisplayName: "IterativePower",
		Signature:   "func IterativePower(nb int, power int) int",
		Solution: `package piscine

func IterativePower(nb int, power int) int {
	if power < 0 {
		return 0
	}
	result := 1
	for i := 0; i < power; i++ {
		result *= nb
	}
	return result
}`,
		Description: "Возводит nb в степень power итеративно. Для отрицательных степеней возвращает 0.",
		Concepts:    []string{"степень", "power", "возведение", "цикл"},
	},
	{
		Name:        "recursivepower",
		DisplayName: "RecursivePower",
		Signature:   "func RecursivePower(nb int, power int) int",
		Solution: `package piscine

func RecursivePower(nb int, power int) int {
	if power < 0 {
		return 0
	}
	if power == 0 {
		return 1
	}
	return nb * RecursivePower(nb, power-1)
}`,
		Description: "Возводит nb в степень power рекурсивно. Базовый случай: power == 0 возвращает 1.",
		Concepts:    []string{"степень рекурсивно", "recursion", "recursive power", "базовый случай"},
	},
	{
		Name:        "sqrt",
		DisplayName: "Sqrt",
		Signature:   "func Sqrt(nb int) int",
		Solution: `package piscine

func Sqrt(nb int) int {
	if nb <= 0 {
		return 0
	}
	for i := 1; i*i <= nb; i++ {
		if i*i == nb {
			return i
		}
	}
	return 0
}`,
		Description: "Возвращает целочисленный квадратный корень из nb если он точный, иначе 0.",
		Concepts:    []string{"корень", "sqrt", "квадратный корень", "perfect square"},
	},
	{
		Name:        "isprime",
		DisplayName: "IsPrime",
		Signature:   "func IsPrime(nb int) bool",
		Solution: `package piscine

func IsPrime(nb int) bool {
	if nb < 2 {
		return false
	}
	for i := 2; i*i <= nb; i++ {
		if nb%i == 0 {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет является ли nb простым числом. Оптимизация: достаточно проверить делители до sqrt(nb).",
		Concepts:    []string{"простое число", "prime", "is prime", "делитель", "проверить простое", "проверка простого", "проверить prime", "является простым"},
	},
	{
		Name:        "findnextprime",
		DisplayName: "FindNextPrime",
		Signature:   "func FindNextPrime(nb int) int",
		Solution: `package piscine

func FindNextPrime(nb int) int {
	if nb <= 2 {
		return 2
	}
	for n := nb; ; n++ {
		if IsPrime(n) {
			return n
		}
	}
}`,
		Description: "Находит ближайшее простое число, большее или равное nb. Использует IsPrime внутри цикла.",
		Concepts:    []string{"следующее простое", "next prime", "find prime", "бесконечный цикл"},
	},

	// ─── Strings: classification, transformation ────────────────────────
	{
		Name:        "isalpha",
		DisplayName: "IsAlpha",
		Signature:   "func IsAlpha(s string) bool",
		Solution: `package piscine

func IsAlpha(s string) bool {
	for _, r := range s {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет что строка состоит только из букв (a-z, A-Z) и цифр (0-9). Пустая строка считается alphanumeric.",
		Concepts:    []string{"alphanumeric", "буквы и цифры", "IsAlpha", "range string", "rune"},
	},
	{
		Name:        "islower",
		DisplayName: "IsLower",
		Signature:   "func IsLower(s string) bool",
		Solution: `package piscine

func IsLower(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет что строка состоит только из строчных латинских букв.",
		Concepts:    []string{"строчные", "lower case", "lowercase", "IsLower"},
	},
	{
		Name:        "isupper",
		DisplayName: "IsUpper",
		Signature:   "func IsUpper(s string) bool",
		Solution: `package piscine

func IsUpper(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет что строка состоит только из заглавных латинских букв.",
		Concepts:    []string{"заглавные", "upper case", "uppercase", "IsUpper"},
	},
	{
		Name:        "isnumeric",
		DisplayName: "IsNumeric",
		Signature:   "func IsNumeric(s string) bool",
		Solution: `package piscine

func IsNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет что строка состоит только из цифр (0-9).",
		Concepts:    []string{"цифры", "numeric", "digits", "IsNumeric"},
	},
	{
		Name:        "isprintable",
		DisplayName: "IsPrintable",
		Signature:   "func IsPrintable(s string) bool",
		Solution: `package piscine

func IsPrintable(s string) bool {
	for _, r := range s {
		if r < ' ' || r == 127 {
			return false
		}
	}
	return true
}`,
		Description: "Проверяет что все символы строки печатаемые (ASCII 32-126).",
		Concepts:    []string{"printable", "печатаемый", "ascii", "control characters"},
	},
	{
		Name:        "tolower",
		DisplayName: "ToLower",
		Signature:   "func ToLower(s string) string",
		Solution: `package piscine

func ToLower(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		result = append(result, r)
	}
	return string(result)
}`,
		Description: "Преобразует все заглавные латинские буквы в строчные. Остальные символы остаются.",
		Concepts:    []string{"в нижний регистр", "to lower", "ToLower", "регистр"},
	},
	{
		Name:        "toupper",
		DisplayName: "ToUpper",
		Signature:   "func ToUpper(s string) string",
		Solution: `package piscine

func ToUpper(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		result = append(result, r)
	}
	return string(result)
}`,
		Description: "Преобразует все строчные латинские буквы в заглавные. Остальные символы остаются.",
		Concepts:    []string{"в верхний регистр", "to upper", "ToUpper", "регистр"},
	},
	{
		Name:        "capitalize",
		DisplayName: "Capitalize",
		Signature:   "func Capitalize(s string) string",
		Solution: `package piscine

func Capitalize(s string) string {
	result := make([]rune, 0, len(s))
	prevIsAlnum := false
	for _, r := range s {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		isAlnum := isLetter || isDigit
		if isLetter && !prevIsAlnum {
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
		} else if isLetter {
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
		}
		result = append(result, r)
		prevIsAlnum = isAlnum
	}
	return string(result)
}`,
		Description: "Делает первую букву каждого слова заглавной, остальные — строчными. Слова разделены не-alphanumeric символами.",
		Concepts:    []string{"capitalize", "первая буква", "title case", "слова"},
	},
	{
		Name:        "strlen",
		DisplayName: "StrLen",
		Signature:   "func StrLen(s string) int",
		Solution: `package piscine

func StrLen(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}`,
		Description: "Возвращает количество символов (рун) в строке. Через range считает руны, не байты.",
		Concepts:    []string{"длина строки", "string length", "StrLen", "rune count"},
	},
	{
		Name:        "strrev",
		DisplayName: "StrRev",
		Signature:   "func StrRev(s string) string",
		Solution: `package piscine

func StrRev(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}`,
		Description: "Разворачивает строку. Конвертирует в []rune чтобы корректно работать с многобайтовыми символами.",
		Concepts:    []string{"reverse", "разворот строки", "StrRev", "rune slice"},
	},
	{
		Name:        "concat",
		DisplayName: "Concat",
		Signature:   "func Concat(str1 string, str2 string) string",
		Solution: `package piscine

func Concat(str1 string, str2 string) string {
	return str1 + str2
}`,
		Description: "Конкатенирует две строки через оператор +.",
		Concepts:    []string{"конкатенация", "concat", "concatenation", "+", "склеить строки"},
	},
	{
		Name:        "compare",
		DisplayName: "Compare",
		Signature:   "func Compare(a, b string) int",
		Solution: `package piscine

func Compare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}`,
		Description: "Лексикографическое сравнение строк. Возвращает -1 если a<b, 1 если a>b, 0 если равны.",
		Concepts:    []string{"compare", "сравнение строк", "лексикографически", "strcmp"},
	},
	{
		Name:        "index",
		DisplayName: "Index",
		Signature:   "func Index(s string, toFind string) int",
		Solution: `package piscine

func Index(s string, toFind string) int {
	if toFind == "" {
		return 0
	}
	for i := 0; i+len(toFind) <= len(s); i++ {
		if s[i:i+len(toFind)] == toFind {
			return i
		}
	}
	return -1
}`,
		Description: "Возвращает индекс первого вхождения подстроки toFind в s. Если не найдена — возвращает -1.",
		Concepts:    []string{"индекс подстроки", "index of", "substring", "найти подстроку", "indexOf"},
	},
	{
		Name:        "firstrune",
		DisplayName: "FirstRune",
		Signature:   "func FirstRune(s string) rune",
		Solution: `package piscine

func FirstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}`,
		Description: "Возвращает первую руну строки. Для пустой строки возвращает 0.",
		Concepts:    []string{"first rune", "первая руна", "первый символ"},
	},
	{
		Name:        "lastrune",
		DisplayName: "LastRune",
		Signature:   "func LastRune(s string) rune",
		Solution: `package piscine

func LastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}`,
		Description: "Возвращает последнюю руну строки. Для пустой строки возвращает 0.",
		Concepts:    []string{"last rune", "последняя руна", "последний символ"},
	},
	{
		Name:        "nrune",
		DisplayName: "NRune",
		Signature:   "func NRune(s string, n int) rune",
		Solution: `package piscine

func NRune(s string, n int) rune {
	if n < 1 {
		return 0
	}
	i := 0
	for _, r := range s {
		i++
		if i == n {
			return r
		}
	}
	return 0
}`,
		Description: "Возвращает n-ю руну строки (1-based). Если n вне диапазона или меньше 1 — возвращает 0.",
		Concepts:    []string{"n-я руна", "nth rune", "по индексу", "rune by index"},
	},
	{
		Name:        "alphacount",
		DisplayName: "AlphaCount",
		Signature:   "func AlphaCount(s string) int",
		Solution: `package piscine

func AlphaCount(s string) int {
	count := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			count++
		}
	}
	return count
}`,
		Description: "Считает количество букв (a-z, A-Z) в строке. Цифры и другие символы игнорируются.",
		Concepts:    []string{"count letters", "счёт букв", "alpha count", "alphabetic"},
	},
	{
		Name:        "basicjoin",
		DisplayName: "BasicJoin",
		Signature:   "func BasicJoin(elems []string) string",
		Solution: `package piscine

func BasicJoin(elems []string) string {
	result := ""
	for _, s := range elems {
		result += s
	}
	return result
}`,
		Description: "Конкатенирует все элементы слайса строк в одну строку без разделителя.",
		Concepts:    []string{"join", "склеить слайс строк", "string slice", "concatenate slice"},
	},
	{
		Name:        "trimatoi",
		DisplayName: "TrimAtoi",
		Signature:   "func TrimAtoi(s string) int",
		Solution: `package piscine

func TrimAtoi(s string) int {
	sign := 1
	signSet := false
	result := 0
	for _, r := range s {
		if !signSet && r == '-' {
			sign = -sign
			continue
		}
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
			signSet = true
		}
	}
	return result * sign
}`,
		Description: "Извлекает все цифры из строки, формирует из них число. Знак минус перед первой цифрой инвертирует знак результата.",
		Concepts:    []string{"trim atoi", "извлечь число", "ascii to int", "parse int", "atoi"},
	},
	{
		Name:        "atoibase",
		DisplayName: "AtoiBase",
		Signature:   "func AtoiBase(s string, base string) int",
		Solution: `package piscine

func AtoiBase(s string, base string) int {
	b := len(base)
	if b < 2 {
		return 0
	}
	// validate base: no duplicates, no + or -
	seen := make(map[rune]bool)
	for _, r := range base {
		if r == '+' || r == '-' || seen[r] {
			return 0
		}
		seen[r] = true
	}
	// digit lookup
	digit := make(map[rune]int, b)
	for i, r := range base {
		digit[r] = i
	}
	result := 0
	for _, r := range s {
		v, ok := digit[r]
		if !ok {
			return 0
		}
		result = result*b + v
	}
	return result
}`,
		Description: "Конвертирует строку s из произвольной системы счисления (определённой base) в десятичное int. Валидирует base.",
		Concepts:    []string{"atoi base", "система счисления", "конвертация", "base conversion", "radix"},
	},
	{
		Name:        "printstr",
		DisplayName: "PrintStr",
		Signature:   "func PrintStr(s string)",
		Solution: `package piscine

import "github.com/01-edu/z01"

func PrintStr(s string) {
	for _, r := range s {
		z01.PrintRune(r)
	}
}`,
		Description: "Печатает строку по одной руне через z01.PrintRune.",
		Concepts:    []string{"print string", "вывод строки", "z01.PrintRune", "печать"},
	},
	{
		Name:        "printcomb",
		DisplayName: "PrintComb",
		Signature:   "func PrintComb()",
		Solution: `package piscine

import "github.com/01-edu/z01"

func PrintComb() {
	for a := '0'; a <= '7'; a++ {
		for b := a + 1; b <= '8'; b++ {
			for c := b + 1; c <= '9'; c++ {
				z01.PrintRune(a)
				z01.PrintRune(b)
				z01.PrintRune(c)
				if a != '7' || b != '8' || c != '9' {
					z01.PrintRune(',')
					z01.PrintRune(' ')
				}
			}
		}
	}
	z01.PrintRune('\n')
}`,
		Description: "Выводит все возрастающие комбинации из 3 разных цифр (012, 013, ..., 789), через запятую с пробелом.",
		Concepts:    []string{"print comb", "комбинации", "тройные циклы", "combinations"},
	},
	{
		Name:        "printcomb2",
		DisplayName: "PrintComb2",
		Signature:   "func PrintComb2()",
		Solution: `package piscine

import "github.com/01-edu/z01"

func PrintComb2() {
	printNum := func(n int) {
		z01.PrintRune(rune('0' + n/10))
		z01.PrintRune(rune('0' + n%10))
	}
	for a := 0; a <= 98; a++ {
		for b := a + 1; b <= 99; b++ {
			printNum(a)
			z01.PrintRune(' ')
			printNum(b)
			if a != 98 || b != 99 {
				z01.PrintRune(',')
				z01.PrintRune(' ')
			}
		}
	}
	z01.PrintRune('\n')
}`,
		Description: "Выводит все пары двузначных чисел (a < b), от 00 01 до 98 99, через запятую и пробел.",
		Concepts:    []string{"print comb 2", "комбинации пар", "двузначные числа"},
	},
	{
		Name:        "printnbrbase",
		DisplayName: "PrintNbrBase",
		Signature:   "func PrintNbrBase(nbr int, base string)",
		Solution: `package piscine

import "github.com/01-edu/z01"

func PrintNbrBase(nbr int, base string) {
	b := len(base)
	if b < 2 {
		return
	}
	seen := make(map[rune]bool)
	for _, r := range base {
		if r == '+' || r == '-' || seen[r] {
			return
		}
		seen[r] = true
	}
	if nbr < 0 {
		z01.PrintRune('-')
		// careful with math.MinInt
		if nbr == -1<<63 {
			nbr++
			defer func() {
				// after recursive call adjust last digit
			}()
		}
		nbr = -nbr
	}
	if nbr >= b {
		PrintNbrBase(nbr/b, base)
	}
	z01.PrintRune(rune(base[nbr%b]))
}`,
		Description: "Печатает число nbr в системе счисления заданной строкой base. Валидирует base и обрабатывает отрицательные числа.",
		Concepts:    []string{"print number base", "система счисления", "вывод в системе", "recursion"},
	},
	{
		Name:        "printnbrinorder",
		DisplayName: "PrintNbrInOrder",
		Signature:   "func PrintNbrInOrder(n int)",
		Solution: `package piscine

import "github.com/01-edu/z01"

func PrintNbrInOrder(n int) {
	if n == 0 {
		z01.PrintRune('0')
		return
	}
	digits := []int{}
	for x := n; x > 0; x /= 10 {
		digits = append(digits, x%10)
	}
	// bubble sort ascending
	for i := 0; i < len(digits); i++ {
		for j := i + 1; j < len(digits); j++ {
			if digits[j] < digits[i] {
				digits[i], digits[j] = digits[j], digits[i]
			}
		}
	}
	for _, d := range digits {
		z01.PrintRune(rune('0' + d))
	}
}`,
		Description: "Выводит цифры числа n в порядке возрастания. Например, 321 → '123', 9087 → '0789'.",
		Concepts:    []string{"sort digits", "сортировка цифр", "in order", "печать цифр"},
	},

	// ─── Slices ─────────────────────────────────────────────────────────
	{
		Name:        "appendrange",
		DisplayName: "AppendRange",
		Signature:   "func AppendRange(min, max int) []int",
		Solution: `package piscine

func AppendRange(min, max int) []int {
	if min >= max {
		return nil
	}
	result := []int{}
	for i := min; i < max; i++ {
		result = append(result, i)
	}
	return result
}`,
		Description: "Возвращает слайс целых чисел от min (включительно) до max (не включительно). Если min >= max, возвращает nil.",
		Concepts:    []string{"append range", "слайс диапазона", "range slice", "append"},
	},
	{
		Name:        "makerange",
		DisplayName: "MakeRange",
		Signature:   "func MakeRange(min, max int) []int",
		Solution: `package piscine

func MakeRange(min, max int) []int {
	if min >= max {
		return nil
	}
	result := make([]int, max-min)
	for i := range result {
		result[i] = min + i
	}
	return result
}`,
		Description: "Возвращает слайс целых от min до max (исключая max). Использует make для предварительного выделения памяти.",
		Concepts:    []string{"make range", "make slice", "preallocate", "слайс с make"},
	},
}
