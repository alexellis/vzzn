package cmd

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/alexellis/vzzn/internal/annotate"
)

const labelPrompt = "Detect every distinct object and person in this image. Respond with ONLY a JSON object of the form {\"objects\":[{\"label\":\"<short noun>\",\"box\":[x0,y0,x1,y1]}]} where box corners are integers normalized to 0-1000 against the image's width and height (x0,y0 = top-left, x1,y1 = bottom-right). No prose, no markdown, no commentary."

// MakeLabel returns the label subcommand: an annotated copy of a single image
// with object boxes and labels.
func MakeLabel() *cobra.Command {
	var out string
	c := &cobra.Command{
		Use:   "label IMAGE",
		Short: "Annotated copy of IMAGE with object boxes and labels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabel(args[0], out)
		},
	}
	c.Flags().StringVarP(&out, "output", "o", "", "output file (default IMAGE.label.png; stdout for '-')")
	return c
}

// runLabel completes a detection prompt, renders the boxes onto the image,
// and writes the annotated result. Box/label summaries go to stderr so the
// image bytes (file or stdout) stay clean.
func runLabel(imgPath, out string) error {
	var buf bytes.Buffer
	// minimal: OCR/label are verbatim/structured tasks; skip thinking tokens.
	if err := completeMulti([]string{imgPath}, selectPrompt(labelPrompt), minimalEffort, &buf, os.Stderr, timeoutDur); err != nil {
		return err
	}
	boxes, err := annotate.Parse(buf.String())
	if err != nil {
		return err
	}
	if len(boxes) == 0 {
		return fmt.Errorf("no objects detected")
	}
	for _, b := range boxes {
		fmt.Fprintf(os.Stderr, "%s [%d,%d,%d,%d]\n", b.Label, b.X0, b.Y0, b.X1, b.Y1)
	}

	raw, err := readImage(imgPath)
	if err != nil {
		return err
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decoding %s: %w", imgPath, err)
	}
	src = annotate.ApplyOrientation(src, annotate.ReadOrientation(raw))
	annotated := annotate.Render(src, boxes)

	var w io.Writer
	if out == "" {
		out = defaultLabelOut(imgPath)
	}
	if out == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if err := png.Encode(w, annotated); err != nil {
		return err
	}
	if out != "-" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", out)
	}
	return nil
}

func defaultLabelOut(imgPath string) string {
	if imgPath == "-" {
		return "-"
	}
	return imgPath + ".label.png"
}
