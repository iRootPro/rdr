# rdr iOS — Design Specification

## Design Philosophy

Минималистичный, чистый интерфейс с фокусом на чтение. Вдохновление: Reeder 5, NetNewsWire, Apple News. Нативный iOS look-and-feel, без кастомных UI-компонентов где можно обойтись стандартными.

Цветовая палитра адаптируется к системной теме (Light / Dark).

---

## Color Palette

### Light Mode
- Background: `#FFFFFF`
- Secondary Background: `#F2F2F7` (iOS system grouped background)
- Text Primary: `#000000`
- Text Secondary: `#8E8E93`
- Accent: `#007AFF` (iOS system blue)
- Unread indicator: `#007AFF`
- Star: `#FF9500` (iOS system orange)
- Feed source tag: `#34C759` (iOS system green)
- Destructive: `#FF3B30`

### Dark Mode
- Background: `#000000`
- Secondary Background: `#1C1C1E`
- Text Primary: `#FFFFFF`
- Text Secondary: `#8E8E93`
- Accent: `#0A84FF`
- Остальные — системные iOS dark-адаптации

---

## Typography

- Navigation title: SF Pro Display, 34pt, Bold
- Section header: SF Pro Text, 13pt, Semibold, uppercase, secondary color
- Article title (list): SF Pro Text, 17pt, Semibold
- Article title (read): SF Pro Text, 17pt, Regular, secondary color
- Body text: SF Pro Text, 17pt, Regular
- Meta text (feed name, time): SF Pro Text, 13pt, Regular, secondary color
- Reader title: SF Pro Display, 22pt, Bold
- Reader body: SF Pro Text, 17pt, Regular, 1.5 line spacing

---

## Screen 1: Feed List (Root)

### Layout
- `NavigationSplitView` на iPad, `NavigationStack` на iPhone
- Top bar: "rdr" title, trailing buttons: `+` (add feed), gear icon (settings)

### Content
Секции в `List` с `.listStyle(.insetGrouped)`:

**Smart Folders** (верхняя секция, без header)
- Каждая строка: SF Symbol icon + название + badge с числом непрочитанных
- Иконки:
  - Inbox → `tray.fill`
  - Today → `calendar`
  - This Week → `calendar.badge.clock`
  - Starred → `star.fill`
- Badge: круглый, accent color, белый текст, только если count > 0

**Категории** (отдельная секция для каждой категории)
- Section header: название категории uppercase
- Каждый фид: `favicon` (16x16, круглый) + название + unread badge
- Favicon загружается по URL `https://www.google.com/s2/favicons?domain=DOMAIN&sz=32`
- Fallback favicon: SF Symbol `dot.radiowaves.up.forward` в accent color

### Interactions
- Tap → открывает Article List
- Swipe left → Delete feed (destructive, с подтверждением)
- Long press → контекстное меню: Edit, Move to folder, Copy URL, Delete
- Pull to refresh → sync all feeds

### Empty State
Если нет фидов:
- Большая иконка `dot.radiowaves.up.forward` (60pt, secondary color)
- "No feeds yet" title
- "Add feeds manually or browse the catalog" subtitle
- Две кнопки: "Add Feed" (primary) и "Browse Catalog" (secondary)

---

## Screen 2: Article List

### Layout
- Navigation title: имя фида/папки
- Toolbar: trailing filter button (menu: All, Unread, Starred)
- Search bar встроенный (`.searchable`)

### Article Row
Каждая строка — вертикальный stack:

```
[●] Article Title That Can Span
    Multiple Lines if Needed
    
    Feed Name · 3h ago · 2 min read    ★
```

- `●` — синий кружок (6pt) для непрочитанных, скрыт для прочитанных
- Title: Semibold для unread, Regular + secondary color для read
- Meta line: feed name (green) + dot + time ago + dot + reading time
- Star: orange `★` в правом углу, только если starred
- Preview (optional): 2 строки description text, secondary color, под meta

### Row height
- Без preview: ~72pt
- С preview: ~100pt

### Interactions
- Tap → открывает Reader
- Swipe right → toggle read/unread (icon: `envelope` / `envelope.open`)
- Swipe left → toggle star (icon: `star` / `star.fill`)
- Long press → Share, Copy URL, Mark read, Star

### Toolbar Filter Menu
Кнопка с SF Symbol `line.3.horizontal.decrease.circle`:
- All articles (checkmark if active)
- Unread only
- Starred only

---

## Screen 3: Article Reader

### Layout
Полноэкранный ScrollView. Toolbar:
- Leading: back button (auto)
- Trailing: share button, overflow menu (star, mark read, open in Safari, copy URL)

### Header Block
Отступ сверху 20pt, padding horizontal 16pt:

```
Article Title in Large Bold Font
That Can Span Multiple Lines

Feed Name · 3 hours ago · 5 min read

───────────────────────────────────
```

- Title: SF Pro Display, 22pt, Bold, primary color
- Meta: 13pt, Regular. Feed name в green, остальное secondary
- Divider: hairline, 8pt padding above/below

### Body
- Padding horizontal 16pt
- Font: SF Pro Text, 17pt, Regular
- Line spacing: 1.5
- Paragraphs: 16pt spacing between
- Links: accent color, underline on tap
- Images: full-width, rounded corners 8pt, async loading с placeholder
- Code blocks: monospace font, secondary background, rounded corners 8pt, horizontal scroll
- Blockquotes: 3pt left border accent color, left padding 12pt, italic
- Tables: horizontal scroll container, alternating row background
- Headings: H2 = 20pt Bold, H3 = 17pt Bold

### Fetch Full Article
Если нет cached body:
- Показать description/content как preview
- Floating button внизу: "Load Full Article" с SF Symbol `arrow.down.doc`
- При нажатии: activity indicator, загрузка через readability
- После загрузки: плавная замена контента

### Bottom Bar (optional, like Reeder)
Thin bottom bar с кнопками:
- `star` / `star.fill` — toggle star
- `envelope.open` / `envelope` — toggle read
- `safari` — open in Safari  
- `square.and.arrow.up` — share
- `translate` — translate (AI)

---

## Screen 4: Settings

### Layout
`NavigationStack` с `Form` (`.formStyle(.grouped)`)

### Sections

**General**
- Language: picker (English / Русский)
- Theme: picker (System / Light / Dark)
- Show Preview: toggle
- Show Images: toggle

**Feeds**
- List of feeds grouped by category
- Tap → edit (name, URL, category)
- Swipe → delete
- `+` button → add feed

**Smart Folders**
- List of smart folders
- Tap → edit (name, query)
- Swipe → delete
- `+` → add with name + query

**AI**
- Provider: picker (Claude / OpenAI / Ollama / Apple Intelligence)
- Endpoint: text field (hidden for Claude)
- API Key: secure text field (hidden for Claude)
- Model: text field

**Auto-refresh**
- Interval: picker (Disabled, 5 min, 15 min, 30 min, 1 hour)
- Background refresh: toggle (iOS background app refresh)

**About**
- Version
- GitHub link
- Rate on App Store

---

## Screen 5: Feed Catalog (Discover)

### Layout
`NavigationStack` с `List` секциями по категориям

### Header
Если первый запуск (onboarding):
```
Welcome to rdr!

A clean RSS reader for iOS.
Pick some feeds to get started.
```

### Category Section
Section header: категория (Tech News, Programming, AI/ML, etc.)

Каждый фид:
```
[○/●] Feed Name
      Short description or URL
```

- `○` — не подписан (secondary color circle)
- `●` — подписан (accent color filled circle)
- Tap → toggle subscription (с анимацией checkmark)

### Bottom
"Done" button (primary, full width) → закрывает каталог, запускает sync

---

## Screen 6: Search

### Layout
Отдельный tab или модальный экран

### Search Bar
- Placeholder: "Search articles..."
- Поддержка query syntax: `title:rust unread newer:1w`
- Под search bar — hint text с примерами синтаксиса (collapsible)

### Results
Тот же формат что Article List, но с highlighted matching text

---

## Tab Bar (Main Navigation)

Для iPhone — bottom tab bar:

1. **Feeds** — `dot.radiowaves.up.forward` — Feed List
2. **Today** — `calendar` — Articles from today
3. **Starred** — `star.fill` — Starred articles
4. **Search** — `magnifyingglass` — Search
5. **Settings** — `gear` — Settings

---

## Animations & Transitions

- List row insert/remove: `.animation(.default)`
- Read/unread toggle: smooth opacity transition on title
- Star toggle: scale bounce on star icon
- Pull to refresh: native `RefreshableModifier`
- Navigation: standard iOS push/pop
- Catalog subscribe: checkmark scale animation

---

## iPad Specific

- `NavigationSplitView` with 3 columns: Feeds | Articles | Reader
- Sidebar always visible in landscape
- Reader fills remaining space
- Keyboard shortcuts: arrow keys, space, r (refresh), s (star), etc.

---

## Accessibility

- Dynamic Type: all text scales with user's preferred size
- VoiceOver: all elements labeled
- Reduce Motion: disable animations when enabled
- Bold Text: respect system setting
- High Contrast: use system semantic colors

---

## Data Model (Local SQLite, same schema as TUI)

```
feeds: id, name, url, category, position, created_at
articles: id, feed_id, title, url, description, content,
          published_at, read_at, starred_at, cached_body, cached_at
settings: key, value
smart_folders: id, name, query, position, created_at
```

Тот же формат что в TUI — при будущем sync будет проще.

---

## First Launch Flow

1. **Language picker** → fullscreen, two big buttons: English / Русский
2. **Catalog** → browse and subscribe to feeds (with welcome message)
3. **Main screen** → sync starts automatically, articles appear
