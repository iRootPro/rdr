# AI: перевод и суммаризация

`rdr` умеет переводить и суммаризировать статьи через OpenAI-совместимые API и CLI-провайдеры.

## Провайдеры

| Provider | Как работает | Endpoint/API Key в rdr | Model |
|---|---|---|---|
| `openai` | HTTP `/chat/completions` | нужны `Endpoint`, опционально `API Key` | обязателен |
| `claude` | `claude --print` | не используются | опционально, если имя начинается с `claude` |
| `pi` | `pi --print` | не используются | опционально |
| `opencode` | `opencode run` | не используются | опционально; если пусто, используется конфигурация opencode |

## Поведение

- `t` — переводит статью на язык интерфейса.
- `Ctrl+s` — суммаризирует статью на языке самой статьи.
  - Русская статья → summary на русском.
  - Английская статья → summary на английском.
  - Если языков несколько, используется преобладающий язык статьи.

## Настройка в TUI

Откройте `Settings` (`s`) → вкладка **AI**:

1. На строке `Provider` нажмите `enter`.
2. Выберите `openai`, `claude`, `pi` или `opencode`.
3. Для CLI-провайдеров `Endpoint` и `API Key` очищаются и игнорируются.
4. `Model` можно оставить пустым, кроме `openai`, где модель обязательна.

## pi

`rdr` запускает pi в безопасном одноразовом режиме:

```bash
pi --print \
  --no-session \
  --no-tools \
  --no-context-files \
  --no-extensions \
  --no-skills \
  --no-prompt-templates \
  --no-themes \
  --system-prompt "..." \
  "текст статьи"
```

То есть для перевода и суммаризации pi используется только как текстовая модель: без доступа к инструментам, файлам проекта, сессиям и внешнему контексту.

Перед использованием настройте pi вне `rdr`, например через переменные окружения/API-ключи, и проверьте:

```bash
pi -p "Say hello"
```

## opencode

`rdr` запускает opencode так:

```bash
opencode run --pure --format default "инструкция + текст статьи"
```

Если поле `Model` в `rdr` пустое, opencode использует свою модель/агента из собственной конфигурации. Если `Model` задан, он передаётся как `--model`, например:

```text
anthropic/claude-sonnet-4
```

Перед использованием настройте opencode вне `rdr` и проверьте:

```bash
opencode run "Say hello"
```

## OpenAI-совместимые API

Для `openai` нужны:

```text
Provider: openai
Endpoint: https://api.openai.com/v1
API Key: sk-...
Model: gpt-4o-mini
```

Этот же режим подходит для локальных OpenAI-compatible серверов, например Ollama или apfel.
