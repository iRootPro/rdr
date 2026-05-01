# AI: Translation and Summarization

`rdr` can translate and summarize articles via OpenAI-compatible APIs and local CLI providers.

## Providers

| Provider | How it works | Endpoint/API Key in rdr | Model |
|---|---|---|---|
| `openai` | HTTP `/chat/completions` | requires `Endpoint`, optional `API Key` | required |
| `claude` | `claude --print` | ignored | optional, only if the name starts with `claude` |
| `pi` | `pi --print` | ignored | optional |
| `opencode` | `opencode run` | ignored | optional; if empty, opencode uses its own configuration |

## Behavior

- `t` translates the article to the UI language.
- `Ctrl+s` summarizes the article in the article's own language.
  - Russian article → Russian summary.
  - English article → English summary.
  - Mixed-language article → the article's predominant language.

## TUI setup

Open `Settings` (`s`) → **AI** tab:

1. Press `enter` on the `Provider` row.
2. Choose `openai`, `claude`, `pi`, or `opencode`.
3. CLI providers clear and ignore `Endpoint` and `API Key`.
4. `Model` may be empty, except for `openai`, where it is required.

## pi

`rdr` runs pi in a safe one-shot mode:

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
  "article text"
```

For translation and summarization, pi is used only as a text model: no tools, project files, sessions, or external context.

Configure pi outside `rdr` first, then verify:

```bash
pi -p "Say hello"
```

## opencode

`rdr` runs opencode like this:

```bash
opencode run --pure --format default "instruction + article text"
```

If `Model` is empty in `rdr`, opencode uses its own configured model/agent. If `Model` is set, it is passed as `--model`, for example:

```text
anthropic/claude-sonnet-4
```

Configure opencode outside `rdr` first, then verify:

```bash
opencode run "Say hello"
```

## OpenAI-compatible APIs

For `openai`, configure:

```text
Provider: openai
Endpoint: https://api.openai.com/v1
API Key: sk-...
Model: gpt-4o-mini
```

This mode also works with local OpenAI-compatible servers, such as Ollama or apfel.
