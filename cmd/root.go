// Package cmd holds the vzzn cobra command tree: the describe command (root)
// with the ocr, label and version subcommands, and the shared completion path.
// Version/build metadata (Version, GitCommit) is injected into this package at
// build time via -ldflags -X — see the Makefile and version.go.
package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexellis/vzzn/internal/auth"
	"github.com/alexellis/vzzn/internal/config"
	"github.com/alexellis/vzzn/internal/gateway"
)

const (
	defaultProvider = "toilgate"
	defaultModel    = "Qwen3.8-27B-FP8-vllm"
	defaultTimeout  = 60 * time.Second

	describePrompt = "Describe this image in detail."

	// minimalEffort sets enable_thinking:false server-side: verbatim/structured
	// tasks skip the thinking block entirely (no latency, no quality cost).
	minimalEffort = "minimal"

	// the gateway parses up to 32MB to extract the model id; keep the
	// encoded body comfortably under that.
	maxImageBytes = 22 << 20
)

var (
	model      string
	provider   string
	reasoning  string
	timeoutDur time.Duration
	stream     bool
	prompt     string
)

// MakeRoot assembles the vzzn command tree: the describe command (root) with
// the ocr, label and version subcommands, and the shared persistent flags.
func MakeRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "vzzn IMAGE [PROMPT ...]",
		Short: "Lean vision/OCR client for an LLM gateway",
		Long: `vzzn sends one multimodal chat completion to an OpenAI-compatible
gateway and streams the answer to stdout. By default endpoints and
credentials come from the local opencode configuration; a ~/.vzzn/config.json
can set url, model and token directly to point at another gateway.

With no subcommand, IMAGE is described (or described with a custom PROMPT).
The ocr subcommand transcribes text literally and accepts one or more
images; the label subcommand writes an annotated copy of a single image
with object boxes and labels.

--prompt overrides the base prompt for any command; on the default
describe command it takes precedence over a positional PROMPT.

~/.vzzn/config.json optionally overrides url, model and token;
~/.vzzn/token.json caches vzzn's own access token when using the
opencode path.`,
		// Override cobra's default validation, which would otherwise reject
		// positional args on a root command that also has subcommands. This
		// lets "vzzn IMAGE" fall through to describe while "vzzn ocr IMAGE"
		// routes to the ocr subcommand.
		Args: cobra.MinimumNArgs(1),
		RunE: describe,
	}
	pf := root.PersistentFlags()
	pf.StringVar(&model, "model", "", "model id")
	pf.StringVar(&provider, "provider", defaultProvider, "opencode config provider")
	pf.StringVar(&reasoning, "reasoning", "", "reasoning_effort override (minimal|low|medium|high|xhigh)")
	pf.DurationVarP(&timeoutDur, "timeout", "t", defaultTimeout, "overall deadline including retries")
	pf.BoolVar(&stream, "stream", true, "stream the answer (disable to buffer the full completion)")
	pf.StringVar(&prompt, "prompt", "", "override the base prompt for the command")
	root.AddCommand(MakeOCR(), MakeLabel(), MakeVersion())
	return root
}

// selectPrompt returns the --prompt override when set, otherwise base.
func selectPrompt(base string) string {
	if prompt != "" {
		return prompt
	}
	return base
}

func describe(cmd *cobra.Command, args []string) error {
	p := describePrompt
	if len(args) >= 2 {
		p = strings.Join(args[1:], " ")
	}
	return run(args[0], selectPrompt(p), "")
}

func run(imgPath, prompt, reasoningDefault string) error {
	return completeMulti([]string{imgPath}, prompt, reasoningDefault, os.Stdout, os.Stderr, timeoutDur)
}

func completeMulti(imgPaths []string, prompt, reasoningDefault string, out, progress io.Writer, timeout time.Duration) error {
	raws := make([][]byte, len(imgPaths))
	var totalRaw int64
	for i, p := range imgPaths {
		raw, err := readImage(p)
		if err != nil {
			return err
		}
		raws[i] = raw
		totalRaw += int64(len(raw))
	}
	if totalRaw*4/3 > maxImageBytes {
		return fmt.Errorf("images total %d bytes; encoded body would exceed the gateway's 32MB request limit — downscale them first", totalRaw)
	}

	local, err := config.LoadLocal()
	if err != nil {
		return err
	}

	var baseURL, token, tokenURL string
	if local.URL != "" {
		baseURL = strings.TrimSuffix(local.URL, "/")
		token = local.Token
	} else {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		baseURL, tokenURL, err = config.Resolve(cfg, provider)
		if err != nil {
			return err
		}
		token, err = auth.Token(tokenURL)
		if err != nil {
			return err
		}
	}

	parts := make([]gateway.Part, 0, len(imgPaths)+1)
	parts = append(parts, gateway.Part{Type: "text", Text: prompt})
	for i, p := range imgPaths {
		parts = append(parts, gateway.Part{
			Type: "image_url",
			ImageURL: &gateway.ImageURL{
				URL: "data:" + mime(p) + ";base64," + base64.StdEncoding.EncodeToString(raws[i]),
			},
		})
	}

	req := gateway.Request{
		Model:  model,
		Stream: stream,
		Messages: []gateway.Message{{
			Role:    "user",
			Content: parts,
		}},
	}
	if req.Model == "" {
		if local.Model != "" {
			req.Model = local.Model
		} else {
			req.Model = defaultModel
		}
	}
	if reasoning != "" {
		req.ReasoningEffort = reasoning
	} else if reasoningDefault != "" {
		req.ReasoningEffort = reasoningDefault
	}

	return gateway.Complete(baseURL, token, req, out, progress, timeout)
}

func readImage(name string) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(name)
}

func mime(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/png"
	}
}
