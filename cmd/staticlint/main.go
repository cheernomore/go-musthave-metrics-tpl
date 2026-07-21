// Command staticlint — это multichecker статических анализаторов для проекта
// go-musthave-metrics-tpl.
//
// # Состав multichecker
//
// Multichecker объединяет несколько групп анализаторов:
//
//  1. Стандартные анализаторы пакета golang.org/x/tools/go/analysis/passes
//     (подмножество проверок go vet): printf, structtag, copylock, lostcancel,
//     httpresponse, loopclosure, nilfunc, unmarshal, unreachable, unsafeptr,
//     unusedresult и другие. Они выявляют распространённые ошибки: неверные
//     форматные строки, копирование мьютексов, утечки context.CancelFunc,
//     незакрытые тела HTTP-ответов, недостижимый код и т. п.
//
//  2. Все анализаторы класса SA пакета staticcheck.io
//     (honnef.co/go/tools/staticcheck). Класс SA — это проверки корректности:
//     потенциальные баги, некорректное использование стандартной библиотеки,
//     гонки, мёртвый код и пр.
//
//  3. Анализаторы остальных классов staticcheck.io:
//     - S  (honnef.co/go/tools/simple)     — упрощение кода;
//     - ST (honnef.co/go/tools/stylecheck) — соответствие стилю и соглашениям;
//     - QF (honnef.co/go/tools/quickfix)   — предложения по рефакторингу.
//
//  4. Публичные сторонние анализаторы:
//     - bodyclose (github.com/timakin/bodyclose) — проверяет, что тело
//     http.Response закрывается;
//     - nilerr (github.com/gostaticanalysis/nilerr) — выявляет возврат nil
//     при ненулевой ошибке (и наоборот).
//
//  5. Собственный анализатор exitcheck — запрещает прямой вызов os.Exit
//     в функции main пакета main (см. пакет exitcheck).
//
// # Запуск
//
// Сборка бинарника:
//
//	go build -o staticlint ./cmd/staticlint
//
// Запуск по всем пакетам проекта:
//
//	./staticlint ./...
//
// Запуск одного анализатора (например, собственного exitcheck) и просмотр
// списка доступных анализаторов и флагов:
//
//	./staticlint -exitcheck ./...
//	./staticlint help
//
// Multichecker возвращает ненулевой код выхода, если хотя бы один анализатор
// нашёл проблему, поэтому его удобно использовать в CI.
package main

import (
	"strings"

	"github.com/cheernomore/go-musthave-metrics-tpl/cmd/staticlint/exitcheck"
	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	multichecker.Main(buildChecks()...)
}

// buildChecks собирает полный набор анализаторов multichecker:
// стандартные passes, все SA staticcheck, классы S/ST/QF, публичные
// анализаторы и собственный exitcheck.
func buildChecks() []*analysis.Analyzer {
	checks := standardPasses()

	// Все анализаторы класса SA из staticcheck.io.
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			checks = append(checks, a.Analyzer)
		}
	}

	// Анализаторы остальных классов staticcheck.io: S, ST и QF.
	for _, a := range simple.Analyzers {
		checks = append(checks, a.Analyzer)
	}
	for _, a := range stylecheck.Analyzers {
		checks = append(checks, a.Analyzer)
	}
	for _, a := range quickfix.Analyzers {
		checks = append(checks, a.Analyzer)
	}

	// Публичные сторонние анализаторы и собственный анализатор.
	checks = append(checks,
		bodyclose.Analyzer,
		nilerr.Analyzer,
		exitcheck.Analyzer,
	)

	return checks
}

// standardPasses возвращает стандартные анализаторы пакета
// golang.org/x/tools/go/analysis/passes.
func standardPasses() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		appends.Analyzer,       // проверяет, что append используется с аргументами
		asmdecl.Analyzer,       // соответствие ассемблерных файлов сигнатурам Go
		assign.Analyzer,        // бессмысленные присваивания вида x = x
		atomic.Analyzer,        // некорректное использование sync/atomic
		bools.Analyzer,         // подозрительные логические выражения
		buildtag.Analyzer,      // корректность build-тегов
		cgocall.Analyzer,       // нарушения правил передачи указателей в cgo
		composite.Analyzer,     // составные литералы без указания полей
		copylock.Analyzer,      // копирование значений, содержащих мьютексы
		defers.Analyzer,        // ошибки в выражениях defer (например, time.Since)
		directive.Analyzer,     // корректность директив //go:
		errorsas.Analyzer,      // неверный второй аргумент errors.As
		httpresponse.Analyzer,  // ошибки при работе с http.Response
		ifaceassert.Analyzer,   // невозможные приведения типов интерфейсов
		loopclosure.Analyzer,   // захват переменной цикла замыканием
		lostcancel.Analyzer,    // потеря функции отмены context.CancelFunc
		nilfunc.Analyzer,       // бессмысленное сравнение функции с nil
		printf.Analyzer,        // соответствие форматных строк аргументам
		shift.Analyzer,         // сдвиги, превышающие разрядность числа
		sigchanyzer.Analyzer,   // небуферизированные каналы для signal.Notify
		slog.Analyzer,          // некорректные пары ключ-значение в log/slog
		stdmethods.Analyzer,    // сигнатуры методов стандартных интерфейсов
		stringintconv.Analyzer, // подозрительные преобразования int в string
		structtag.Analyzer,     // корректность тегов структур
		tests.Analyzer,         // ошибки в сигнатурах тестов и примеров
		timeformat.Analyzer,    // неверные форматы времени
		unmarshal.Analyzer,     // передача не-указателя в Unmarshal
		unreachable.Analyzer,   // недостижимый код
		unsafeptr.Analyzer,     // некорректные преобразования в unsafe.Pointer
		unusedresult.Analyzer,  // игнорирование результата чистых функций
	}
}
