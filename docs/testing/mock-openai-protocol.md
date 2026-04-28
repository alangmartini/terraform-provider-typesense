# Mock OpenAI Protocol (Typesense LLM validation)

Empirical findings from Task 0.1 in `tasks/plan.md`. Used by the chinook E2E suite to mock LLM endpoints when creating `nl_search_models` / `conversation_models`.

## Setup

- Typesense `30.1` container with `--add-host=host.docker.internal:host-gateway` (works on Docker Desktop and Linux).
- Mock HTTP server bound to `0.0.0.0:<port>` on the host.
- Reach the host from the container at `http://host.docker.internal:<port>`.

## What Typesense sends

When you `POST /nl_search_models` with `api_url` set, Typesense issues a validation request **before** persisting the model:

| Aspect | Value |
|--------|-------|
| Method | `POST` |
| URL | The `api_url` value, **as-is** (no path appended) |
| Body | `{"max_tokens":10,"messages":[{"content":"hello","role":"user"}],"model":"<stripped>","temperature":0}` |
| Headers | `Content-Type: application/json`, `Authorization: Bearer <api_key>` (presumed; not yet captured) |

Notes:
- `model_name` is sent stripped of its `vllm/` or `openai/` prefix. `vllm/test` becomes `test`.
- The mock responds with any 2xx + valid OpenAI-shaped chat-completion JSON; Typesense accepts and persists the model.

## Behavior matrix

| `model_name` prefix | `api_url` | Outcome |
|---------------------|-----------|---------|
| `vllm/*`            | provided  | Typesense calls the api_url, accepts model on 2xx. |
| `vllm/*`            | missing   | 400: "Property `api_url` is missing or is not a non-empty string." |
| `openai/*`          | provided  | Typesense calls the api_url (override is honored, not just for vllm). |
| `openai/*`          | missing   | Typesense calls `https://api.openai.com/v1/chat/completions` (real OpenAI). |

The third row is the key finding: **chinook does not need to switch model names to `vllm/*`**. Setting `api_url` on the existing `openai/gpt-4o-mini` resources is enough to redirect to the mock during tests.

## Required mock response

```json
{
  "id": "chatcmpl-mock-1",
  "object": "chat.completion",
  "created": 1714060000,
  "model": "test",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "{\"q\":\"*\"}"},
      "finish_reason": "stop"
    }
  ],
  "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
}
```

Typesense doesn't appear to inspect the body content closely for nl_search_models creation — any well-formed chat-completion JSON works.

## Implications for the harness

1. The mock OpenAI helper exposes a single POST handler at any path; tests pass `mock.URL` directly as `api_url`.
2. Chinook's `nl_search_model` and `conversation_model` resources gain a `mock_openai_url` variable. When set (chinook-e2e tests), it overrides `api_url`. When empty (real-cluster usage), Typesense falls through to the real OpenAI endpoint.
3. No model_name swap needed — keep `openai/gpt-4o-mini` defaults.
