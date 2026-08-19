# vzzn

A lean vision/OCR client for toilgate, OpenFaaS Ltd's internal LLM
gateway. Sends one multimodal chat completion and streams the answer to
stdout. The code is written to be adapted to other gateways or inference
providers as required by an AI agent.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/alexellis/vzzn/master/get.sh | sh
```

Requires an opencode config at `~/.config/opencode/opencode.json` with a
`toilgate` provider and a stored toilgate auth token.

## Usage

```bash
vzzn IMAGE [PROMPT ...]   # describe
vzzn ocr IMAGE [IMAGE...] # verbatim transcription
vzzn label IMAGE [-o OUT] # annotated copy with object boxes
vzzn version
```

Streaming is on by default; disable with `--stream=false`. Model defaults to
`Qwen3.8-27B-FP8-vllm` (override via `--model` or `~/.vzzn/config.json`).

## Build

```bash
make test
make dist
```

## License

MIT
