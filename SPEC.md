# rdr — TUI RSS Reader

## Концепция

`rdr` — терминальный RSS-ридер нового поколения. В отличие от существующих решений (newsboat и др.),
читать статьи можно прямо внутри приложения: чистый текст, инлайн-картинки через Kitty Graphics
Protocol, без рекламы и без перехода в браузер.

---

## Стек

| Слой | Технология |
|---|---|
| Язык | Go 1.22+ |
| TUI framework | `charmbracelet/bubbletea` |
| Компоненты | `charmbracelet/bubbles` (viewport, spinner, textinput) |
| Стили | `charmbracelet/lipgloss` |
| Markdown рендер | `charmbracelet/glamour` |
| RSS парсинг | `mmcdole/gofeed` |
| Извлечение статей | `go-shiori/go-readability` |
| HTML → Markdown | `JohannesKaufmann/html-to-markdown` |
| Изображения | Kitty Graphics Protocol (raw escape sequences) |
| БД | `mattn/go-sqlite3` |
| Конфиг | YAML (`gopkg.in/yaml.v3`) — только для начального импорта, далее всё через TUI |

---

## UX и навигация

### Основной флоу

```
┌─────────────────────────────────────────────────────┐
│  Feed List  │  Article List                         │
│             │                                       │
│  > HN    5  │  > Заголовок статьи          2h ago   │
│    Lobsters │    Ещё одна статья           3h ago   │
│    Go Blog  │    ...                                │
│             │                                       │
└─────────────────────────────────────────────────────┘
```

- Левая панель — список фидов со счётчиком непрочитанных
- Правая панель — список статей выбранного фида
- `enter` на статье → Reader
- `s` → Settings overlay

### Reader

```
┌─────────────────────────────────────────────────────┐
│  Заголовок статьи                                   │
│  Go Blog · 2h ago · https://...                     │
│  ──────────────────────────────────────────────     │
│                                                     │
│  Текст из RSS-фида (быстро, сразу)                  │
│                                                     │
│  [f] загрузить полную версию                        │
└─────────────────────────────────────────────────────┘
```

**Два режима ридера:**
1. **По умолчанию** — показывает контент из RSS-фида (мгновенно)
2. **Full fetch** (`f`) — загружает полную страницу через go-readability,
   рендерит чистый Markdown с инлайн-картинками через Kitty Graphics Protocol

### Settings (`s`)

Отдельный overlay поверх основного экрана:
- Добавить фид (название + URL)
- Удалить фид
- Изменить название фида
- Общие настройки (интервал обновления, макс. кол-во статей и т.д.)
- Всё сохраняется в SQLite, не в файл

---

## Клавиши

### Глобальные

| Клавиша | Действие |
|---|---|
| `q` | выход |
| `s` | открыть Settings |
| `r` | обновить текущий фид |
| `R` | обновить все фиды |
| `?` | help |

### Навигация (Feed List и Article List)

| Клавиша | Действие |
|---|---|
| `j` / `k` | вниз / вверх |
| `g` / `G` | начало / конец |
| `^d` / `^u` | страница вниз / вверх |
| `l` / `enter` | перейти вправо / открыть |
| `h` / `esc` | перейти влево / назад |
| `tab` | переключить фокус между панелями |

### Reader

| Клавиша | Действие |
|---|---|
| `j` / `k` | скролл вниз / вверх |
| `^d` / `^u` | страница |
| `g` / `G` | начало / конец |
| `f` | загрузить полную статью |
| `o` | открыть в браузере |
| `esc` / `h` | вернуться к списку статей |

### Settings

| Клавиша | Действие |
|---|---|
| `a` | добавить фид |
| `d` | удалить выбранный фид |
| `e` | редактировать выбранный фид |
| `esc` / `s` | закрыть settings |

---

## Архитектура

```
rdr/
├── main.go
├── config.yaml                    ← только для первичного импорта фидов
└── internal/
    ├── config/
    │   └── config.go              ← загрузка YAML (только при первом запуске)
    ├── db/
    │   ├── db.go                  ← инициализация SQLite, миграции
    │   ├── feeds.go               ← CRUD фидов
    │   ├── articles.go            ← CRUD статей, статус read/unread
    │   └── settings.go            ← хранение настроек
    ├── feed/
    │   ├── fetcher.go             ← fetch RSS, параллельно по всем фидам
    │   └── reader.go              ← go-readability + HTML→Markdown
    ├── kitty/
    │   └── image.go               ← Kitty Graphics Protocol, рендер инлайн-картинок
    └── ui/
        ├── model.go               ← главный bubbletea model, state machine
        ├── styles.go              ← lipgloss стили, цветовая тема
        ├── keys.go                ← все key bindings
        ├── messages.go            ← async tea.Msg типы
        ├── feedlist.go            ← компонент левой панели
        ├── articlelist.go         ← компонент правой панели
        ├── reader.go              ← компонент ридера
        └── settings.go            ← settings overlay
```

### SQLite схема

```sql
CREATE TABLE feeds (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    position    INTEGER NOT NULL DEFAULT 0,   -- порядок в списке
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE articles (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL,
    description  TEXT,
    content      TEXT,                        -- HTML из RSS
    published_at DATETIME,
    read_at      DATETIME,                    -- NULL = непрочитана
    cached_at    DATETIME,                    -- NULL = не кэширована
    cached_body  TEXT,                        -- Markdown после readability
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Дефолтные настройки
INSERT INTO settings VALUES ('refresh_interval', '30');
INSERT INTO settings VALUES ('max_articles_per_feed', '50');
INSERT INTO settings VALUES ('theme', 'dark');
```

---

## Kitty Graphics Protocol

Изображения рендерятся **инлайн в тексте статьи**, на том месте где они стоят в оригинале.

**Пайплайн:**
1. go-readability возвращает чистый HTML
2. Парсим `<img>` теги, запоминаем их позиции
3. Скачиваем картинки асинхронно
4. Кодируем в base64, отправляем через Kitty escape sequence
5. В viewport вставляем placeholder нужной высоты на месте картинки

**Fallback** для не-Kitty терминалов: показываем `[image: alt text]` вместо картинки.

Определение Kitty: проверяем `$TERM == "xterm-kitty"` или `$KITTY_WINDOW_ID`.

---

## Дизайн и тема

Тёмная тема, Tokyo Night palette:

```
Background:  #1a1b26
Alt BG:      #24283b
Border:      #3b4261
Muted:       #565f89
Text:        #c0caf5
Accent:      #7aa2f7  (синий  — активные элементы, выделение)
Secondary:   #bb9af7  (фиолетовый — выбранный фид)
Green:       #9ece6a  (источник, счётчики)
Orange:      #ff9e64  (время)
Red:         #f7768e  (ошибки)
Yellow:      #e0af68  (непрочитанные)
Teal:        #2ac3de  (ссылки, URL)
```

---

## Фазы разработки

### Phase 1 — Core
- Структура проекта, go.mod, SQLite инициализация
- Feed List + Article List (split-pane layout)
- Async RSS fetching через gofeed
- Базовый Reader (контент из RSS, без fetch)
- Счётчики непрочитанных, persistence в SQLite
- Vim-биндинги, цветовая тема
- Status bar со спиннером

### Phase 2 — Full Reader
- go-readability: fetch полной статьи по `f`
- HTML → Markdown конвертация
- Glamour рендер с темой
- Kitty Graphics Protocol: инлайн-картинки в тексте

### Phase 3 — Settings TUI
- Settings overlay (`s`)
- Добавление / удаление / редактирование фидов через TUI
- Управление настройками (интервал, лимиты)
- Импорт из config.yaml при первом запуске

### Phase 4 — Кэш и поиск
- Кэширование статей в SQLite (`cached_body`)
- Офлайн-чтение кэшированных статей
- Поиск `/` по заголовкам и тексту статей
- OPML импорт/экспорт
- Фильтры (только непрочитанные, по дате)

---

## Запуск и установка

```bash
# Dev
go run .

# Build
go build -o rdr .

# Install
go install github.com/yourname/rdr@latest
```

Конфиг и БД хранятся в `~/.config/rdr/`:
```
~/.config/rdr/
├── config.yaml     ← только для первичного импорта
└── rdr.db          ← SQLite, всё остальное
```
