package kit

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShimmerSpansLength(t *testing.T) {
	hi := ui.Hex("#111111")
	dim := ui.Hex("#888888")
	sp := shimmerSpans("abcdefghij", hi, dim, .5)
	if len(sp) == 0 || len(sp) > shimmerBins {
		t.Fatalf("bins %d", len(sp))
	}
	if strings.Join(spanTexts(sp), "") != "abcdefghij" {
		t.Fatal("lost text")
	}
}

func TestShimmerBandPosPause(t *testing.T) {
	SetMotion(true)
	p := shimmerBandPos()
	if p < 0 || p > 1 {
		t.Fatalf("pos %v", p)
	}
}

func spanTexts(s []ui.Span) []string {
	out := make([]string, len(s))
	for i, x := range s {
		out[i] = x.Text
	}
	return out
}
