# rdr Phase 1 — Design Document

**Date:** 2026-04-13
**Scope:** Phase 1 (Core) из `SPEC.md`
**Status:** validated, ready for implementation planning

---

## Контекст

`rdr` — терминальный RSS-ридер на Go с split-pane UI (bubbletea), SQLite
хранилищем и встроенным чтением статей. Полная концепция и клавиши описаны
в `SPEC.md`. Этот документ фиксирует решения для Phase 1 — Core MVP.

**Что входит в Phase 1:**
- Структура проекта, SQLite схема и миграции
- Парсинг RSS (gofeed), async fetch всех фидов
- Split-pane UI (FeedList + ArticleList) с Tokyo Night темой
- Базовый Reader (контент из RSS, без full-fetch)
- Счётчики непрочитанных, persistence, vim-биндинги, статусбар

**Что НЕ входит:** go-readability, glamour, Kitty Graphics, Settings TUI,
кэш, поиск, OPML. Это Phase 2–4 по спеке.

---

## Ключевые решения

| Решение | Выбор | Почему |
|---|---|---|
| Порядок работы | Дробим Phase 1 на 6 микро-этапов | Каждый шаг проверяем, малый риск увязнуть |
| Импорт фидов на Phase 1 | `config.yaml` синкается при каждом старте (upsert, без удалений) | Settings TUI ещё нет, надо как-то добавлять фиды |
| Путь к БД | env `RDR_HOME`, дефолт `~/.config/rdr/` | Изолированный dev state без CLI флагов |
| Тестирование | TDD на логику (db, fetcher, config), UI руками | TUI snapshot-тесты хрупкие, дают мало пользы |
| Go module | `github.com/iRootPro/rdr` | Совпадает с путём репо, готово к `go install` |
| Тема | Tokyo Night с первого UI-шага | Переделывать стили в конце = трогать всё заново |

---

## Структура проекта

```
rdr/
├── main.go
├── go.mod / go.sum
├── config.yaml                  ← пример, коммитится
├── SPEC.md
├── docs/plans/
└── internal/
    ├── config/
    │   ├── config.go            ← ResolveHome, Load
    │   └── sync.go              ← Sync(db, cfg)
    ├── db/
    │   ├── db.go                ← Open, миграции
    │   ├── feeds.go             ← CRUD фидов
    │   ├── articles.go          ← CRUD статей, TrimArticles
    │   └── settings.go          ← key-value
    ├── feed/
    │   └── fetcher.go           ← gofeed обёртка, FetchAll с errgroup
    └── ui/
        ├── model.go             ← главный bubbletea Model
        ├── styles.go            ← Tokyo Night палитра
        ├── keys.go              ← KeyMap
        ├── messages.go          ← async tea.Msg типы
        ├── feedlist.go
        ├── articlelist.go
        └── reader.go
```

`.gitignore`: `rdr`, `*.db`, `dev/`, `.env`

**Зависимости Phase 1:**
`charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`,
`mmcdole/gofeed`, `mattn/go-sqlite3`, `gopkg.in/yaml.v3`,
`golang.org/x/sync/errgroup`.

`glamour`, `go-readability`, `html-to-markdown` — в Phase 2.

---

## Слой данных (SQLite)

### Миграция 001

Точно по спеке: таблицы `feeds`, `articles`, `settings`. Индексы:
`idx_articles_feed_id`, `idx_articles_published_at DESC`.

Seed в `settings`: `refresh_interval=30`, `max_articles_per_feed=50`,
`theme=dark`.

Миграции — массив строк в коде, применяются в транзакции, таблица
`schema_migrations(version INTEGER)` отслеживает применённые.

### API

**`db.DB`** — обёртка над `*sql.DB`:

```go
func Open(path string) (*DB, error)   // включает PRAGMA foreign_keys=ON

// feeds.go
func (*DB) ListFeeds() ([]Feed, error)            // с UnreadCount через JOIN
func (*DB) UpsertFeed(name, url string) (Feed, error)
func (*DB) DeleteFeed(id int64) error
func (*DB) GetFeedByURL(url string) (*Feed, error)

// articles.go
func (*DB) ListArticles(feedID int64, limit int) ([]Article, error)
func (*DB) UpsertArticle(a Article) error         // по (feed_id, url), не тронет read_at
func (*DB) MarkRead(id int64) error
func (*DB) TrimArticles(feedID int64, max int) error  // удаляет старые прочитанные

// settings.go
func (*DB) GetSetting(key string) (string, error)
func (*DB) SetSetting(key, value string) error
```

Domain типы `Feed`, `Article` — плоские структуры в `internal/db/`, без ORM.

---

## Конфиг и YAML sync

**`config.Config`:**

```go
type Config struct {
    Feeds []FeedEntry `yaml:"feeds"`
}
type FeedEntry struct {
    Name string `yaml:"name"`
    URL  string `yaml:"url"`
}
```

**Функции:**
- `ResolveHome() (string, error)` — `$RDR_HOME` или `~/.config/rdr/`, создаёт
  директорию при отсутствии.
- `Load(home string) (*Config, error)` — читает `home/config.yaml`. Отсутствие
  файла → пустой `Config{}`, не ошибка.
- `Sync(db *db.DB, cfg *Config) error` — upsert всех записей, фиды
  отсутствующие в YAML **не удаляем**. `position` назначаем по порядку только
  новым фидам.

Пример `config.yaml` коммитим в репо (HN, Go Blog, Lobsters).

---

## Fetcher

```go
type Fetcher struct {
    parser *gofeed.Parser
    db     *db.DB
    client *http.Client   // Timeout: 15s, UA: "rdr/0.1 (+https://github.com/iRootPro/rdr)"
}

func New(db *db.DB) *Fetcher

func (*Fetcher) FetchOne(ctx context.Context, feed db.Feed) (FetchResult, error)
func (*Fetcher) FetchAll(ctx context.Context) ([]FetchResult, error)
```

- `FetchAll` параллелит через `errgroup.WithContext` с семафором на 8 горутин.
- Ошибка одного фида **не прерывает остальных** — кладётся в `Result.Err`.
- После апсерта — `TrimArticles(feedID, max_articles_per_feed)`, удаляет
  только **прочитанные** сверх лимита.
- Маппинг gofeed.Item → db.Article:
  - `Title` — `item.Title`, пусто → `"(без заголовка)"`
  - `URL` — `item.Link`
  - `Content` — `item.Content` или `item.Description`
  - `PublishedAt` — `item.PublishedParsed`, nil → `time.Now()`

Phase 1 — без `If-Modified-Since`/`ETag`, это Phase 4 с кэшем.

---

## UI модель и state machine

**`ui.Model`:**

```go
type focus int
const (
    focusFeeds focus = iota
    focusArticles
    focusReader
)

type Model struct {
    db       *db.DB
    fetcher  *feed.Fetcher

    feeds    []db.Feed
    articles []db.Article
    selFeed  int
    selArt   int
    focus    focus

    reader   readerState
    status   statusState
    width    int
    height   int
}
```

**State machine:**

```
  FeedList ──enter/l──▶ ArticleList ──enter/l──▶ Reader
      ▲                      │                     │
      └────── esc/h ─────────┴────── esc/h ────────┘
```

`tab` переключает фокус между FeedList и ArticleList без ухода в Reader.

**Async tea.Msg:**

```go
type feedsLoadedMsg    struct{ feeds []db.Feed }
type articlesLoadedMsg struct{ feedID int64; articles []db.Article }
type fetchStartedMsg   struct{}
type fetchDoneMsg      struct{ results []feed.FetchResult }
type errMsg            struct{ err error }
type tickMsg           time.Time    // для спиннера / затухания ошибок
```

**Команды:**
- `loadFeedsCmd(db)`
- `loadArticlesCmd(db, feedID)`
- `fetchAllCmd(fetcher)`
- `markReadCmd(db, articleID)`

`Init()` запускает `loadFeedsCmd` + `fetchAllCmd`.

---

## Компоненты UI

### styles.go — Tokyo Night палитра

Константы цветов точно по спеке + готовые `lipgloss.Style`:
`PaneActive`, `PaneInactive`, `ItemSelected`, `Unread`, `Read`, `Counter`,
`Source`, `TimeAgo`, `URLStyle`, `ErrStyle`, `StatusBar`.

### FeedList

- Заголовок "Feeds" в `ColorAccent`.
- Строка фида: `> HN    5` (имя + счётчик справа через `lipgloss.PlaceHorizontal`).
- Счётчик в `Counter` (зелёный), 0 не показываем.
- Выбранный — `ItemSelected`, приглушённый когда `focus != focusFeeds`.
- Фиды с ошибкой фетча — `●` красный префикс.
- Виртуализация: окно строк `[offset, offset+height]`.

### ArticleList

- Строка: заголовок (truncate `…`) + `time ago` справа (оранжевый).
- Непрочитанные — жёлтый, прочитанные — приглушённый.
- `time ago`: `<1h → "Xm ago"`, `<24h → "Xh ago"`, `<7d → "Xd ago"`,
  иначе `"Jan 2"`.

### Reader

- `bubbles/viewport`.
- Шапка: title (accent bold), meta (`Source · TimeAgo · URLStyle`), разделитель.
- Тело: `item.Content` → простой strip HTML через regexp (Phase 1 MVP).
  В Phase 2 заменяем на glamour.
- Подсказка `[f] загрузить полную версию (Phase 2)` — пока неактивна.

### Status bar

Одна строка внизу всегда:
`rdr · 3/12 unread · ⠋ fetching...` или `rdr · 3/12 unread · last: 2m ago`.
Ошибки — `ErrStyle`, затухают через 5s (`tickMsg`).

---

## Порядок разработки

### Шаг 1 — Скелет + БД  *(TDD)*
- `go mod init github.com/iRootPro/rdr`, `.gitignore`
- `internal/config/config.go` (ResolveHome, Load)
- `internal/db/` с миграцией 001 и CRUD
- Тесты: миграции идемпотентны, CRUD фидов/статей, `UpsertArticle` не трогает
  `read_at`, `CountUnread`/UnreadCount в `ListFeeds`
- **AC:** `go test ./...` зелёный, `go build` проходит

### Шаг 2 — YAML sync  *(TDD)*
- `internal/config/sync.go` с `Sync(db, cfg)`
- Пример `config.yaml` в репо
- `main.go`: ResolveHome → db.Open → Load → Sync → print feeds → exit
- **AC:** `RDR_HOME=./dev go run .` создаёт `dev/rdr.db`, синкает, печатает.
  Повторный запуск не дублирует. Идемпотентный sync в тестах.

### Шаг 3 — Fetcher  *(TDD)*
- `internal/feed/fetcher.go` с `FetchOne`/`FetchAll`
- `db.TrimArticles`
- Тесты: `httptest.Server` с Atom/RSS фикстурой, `UpsertArticle` не дублирует,
  битый XML возвращает ошибку, `FetchAll` продолжает при ошибке одного фида
- Временный dev-флаг `--fetch` в main
- **AC:** `go run . --fetch` фетчит и печатает `Added/Updated` по фидам

### Шаг 4 — UI скелет: split-pane  *(руками)*
- `styles.go`, `keys.go`, `messages.go`, `model.go`
- `feedlist.go`, `articlelist.go` — рендер, vim-навигация, tab, enter/l/esc/h
- `main.go` запускает `tea.NewProgram(ui.New(db, fetcher))`, `--fetch` удаляем
- Автофетч при `Init()`, статусбар со спиннером (`bubbles/spinner`)
- `r` — текущий фид, `R` — все, `q` — выход
- **AC:** 2 панели, данные из БД, счётчики живые, навигация работает

### Шаг 5 — Reader  *(руками)*
- `reader.go` с `bubbles/viewport`
- Enter на статье → reader → title/meta/content + скролл
- Открытие → `markReadCmd`, возврат → reload feeds+articles для счётчиков
- Примитивный HTML strip через regexp
- **AC:** скролл работает, счётчики обновляются, статья помечена прочитанной

### Шаг 6 — Polish  *(частично)*
- `?` help оверлей (`bubbles/help`)
- `o` — открыть URL в браузере (`os/exec`: `open`/`xdg-open`)
- Ресайз терминала (`tea.WindowSizeMsg`)
- Пустые состояния, индикатор ошибок у фидов
- **AC:** визуальная проверка сценариев, размеры 80×24 … 200×60

---

## Коммиты

Каждый шаг — один или несколько коммитов. Шаги 1–3 с тестами, шаги 4–6 — с
ручной визуальной проверкой. Между шагами — возможность остановиться и
отрефлексировать.
