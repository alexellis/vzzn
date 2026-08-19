# vzzn

A lean vision/OCR client for an OpenAI-compatible LLM gateway. Sends one
multimodal chat completion and streams the answer to stdout. The code is
written to be adapted to other gateways or inference providers as
required by an AI agent.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/alexellis/vzzn/master/get.sh | sh
```

## Usage

```bash
vzzn IMAGE [PROMPT ...]   # describe
vzzn ocr IMAGE [IMAGE...] # verbatim transcription
vzzn label IMAGE [-o OUT] # annotated copy with object boxes
vzzn version
```

Streaming is on by default; disable with `--stream=false`. Model defaults to
`Qwen3.8-27B-FP8-vllm` (override via `--model`).

## Configuration

By default endpoints come from `~/.config/opencode/opencode.json` (the
`toilgate` provider) and credentials from opencode's stored auth token,
read as a read-only seed. `~/.vzzn/config.json` overrides these:

```json
{
  "url": "https://gateway.example.com/v1",
  "model": "my-model",
  "token": "my-token"
}
```

- `url` and `token`, when set, take over entirely from opencode's config
  and auth store; the endpoint must end in `/v1`.
- `model` overrides the default `Qwen3.8-27B-FP8-vllm`.
- Leave a field out to keep the default behaviour (opencode config + auth).

## Build

```bash
make test
make dist
```

## License

MIT
