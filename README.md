# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки и профилирование памяти

Бенчмарки покрывают ключевые компоненты системы:

- `internal/handler` — обработчики `Updates`, `Index`, `Value`;
- `internal/repository` — `MemStorage.SaveBatch`, `Save`, `FindAll`;
- `cmd/agent` — `convertToModel`, `compressData`, сбор метрик `getMetricsPooler`;
- `cmd/server` — `BenchmarkServerHotPath`: горячий путь сервера (приём батча метрик `POST /updates/` и отдача HTML-страницы `GET /`) на реалистичном наборе из 29 метрик. Этот бенчмарк используется для снятия профиля памяти.

Запуск всех бенчмарков:

```
go test -bench=. -benchmem ./...
```

### Снятие профиля потребления памяти

Профиль снимается с серверного бенчмарка при фиксированном числе итераций
(чтобы `base` и `result` были сопоставимы):

```
go test -run='^$' -bench=BenchmarkServerHotPath -benchtime=5000x \
    -memprofile=profiles/base.pprof ./cmd/server/
```

Анализ профиля (`profiles/base.pprof`) показал, что **72.6%** всех аллокаций
приходится на обработчик `Index`, который на каждый запрос заново разбирал и
компилировал HTML-шаблон (`template.New("index").Parse(...)` плюс компиляция
html-экранирования при первом `Execute`).

### Оптимизация

1. **Шаблон страницы `Index` компилируется один раз** при инициализации пакета
   (`var indexTemplate = template.Must(...)`) вместо разбора на каждый запрос.
2. **Срез имён метрик для аудита формируется лениво** — только когда есть
   хотя бы один приёмник аудита (`HasObservers()`), а не на каждом запросе.

После оптимизации профиль снят повторно в `profiles/result.pprof` той же командой.

### Результат

Суммарные аллокации (`alloc_space`) по бенчмарку:

| Метрика         | base        | result      | Δ              |
|-----------------|-------------|-------------|----------------|
| `alloc_space`   | 357.79 MB   | 226.23 MB   | −131.6 MB (−37%) |
| `alloc_objects` | 6 954 933   | 4 937 689   | −2 017 244 (−29%) |
| на операцию     | 71826 B/op  | 50429 B/op  | −30%           |

Вывод команды сравнения профилей:

```
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
File: server.test
Type: alloc_space
Showing nodes accounting for -76.06MB, 21.26% of 357.79MB total
Dropped 43 nodes (cum <= 1.79MB)
Showing top 12 nodes out of 121
      flat  flat%   sum%        cum   cum%
  -11.51MB  3.22%  3.22%   -11.51MB  3.22%  text/template.addFuncs
  -11.51MB  3.22%  6.43%   -11.51MB  3.22%  maps.Copy[...html/template.context...] (inline)
  -10.51MB  2.94%  9.37%   -10.51MB  2.94%  text/template.addValueFuncs
      -9MB  2.52% 11.89%       -9MB  2.52%  html/template.makeEscaper (inline)
   -8.52MB  2.38% 14.27%    -8.52MB  2.38%  bytes.growSlice
   -8.50MB  2.38% 16.64%       -6MB  1.68%  reflect.Value.call
    7.50MB  2.10% 14.55%        4MB  1.12%  reflect.MakeSlice
   -6.01MB  1.68% 16.23%    -6.01MB  1.68%  reflect.growslice
   -5.51MB  1.54% 17.76%    -5.51MB  1.54%  text/template.builtins
   -5.50MB  1.54% 19.30%    -5.50MB  1.54%  text/template/parse.(*ListNode).append
   -3.50MB  0.98% 20.28%   -32.01MB  8.95%  html/template.(*escaper).escapeTemplateBody
   -3.50MB  0.98% 21.26%    -3.50MB  0.98%  html/template.(*escaper).editActionNode
```

Отрицательные значения подтверждают снижение потребления памяти: исчезли
аллокации разбора и компиляции шаблона (`text/template.*`, `html/template.*`,
`maps.Copy[...context...]`).
