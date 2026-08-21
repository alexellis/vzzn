package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const ocrPrompt = "Transcribe all text visible in this image literally and verbatim, preserving layout where sensible. Output only the transcription."

// MakeOCR returns the ocr subcommand: verbatim transcription of one or more
// images, sent as a single combined multimodal message.
func MakeOCR() *cobra.Command {
	return &cobra.Command{
		Use:   "ocr IMAGE [IMAGE...]",
		Short: "Verbatim transcription of one or more images",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return completeMulti(args, selectPrompt(ocrPrompt), minimalEffort, os.Stdout, os.Stderr, timeoutDur)
		},
	}
}
