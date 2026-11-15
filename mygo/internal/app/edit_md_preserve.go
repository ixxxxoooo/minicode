package app

import (
	"strings"
)

// mdPreserveRoundTrip 在所见即所得序列化后,对照原文保守补回围栏语言标注(一一对应时)。
func mdPreserveRoundTrip(origFull, newFull string) string {
	_, origBody := mdSplitFront(origFull)
	newFront, newBody := mdSplitFront(newFull)
	patched := mdPatchFenceLangs(origBody, newBody)
	if patched == newBody {
		return newFull
	}
	out := newFront
	if patched != "" {
		out += patched
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
	}
	return out
}

func mdPatchFenceLangs(orig, neu string) string {
	olangs := fenceLangs(orig)
	nlangs := fenceLangs(neu)
	if len(olangs) == 0 || len(olangs) != len(nlangs) {
		return neu
	}
	var b strings.Builder
	oi := 0
	fence := ""
	for _, ln := range strings.Split(neu, "\n") {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			b.WriteString(ln)
			b.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = trim[:3]
			lang := ""
			if oi < len(nlangs) && nlangs[oi] == "" && oi < len(olangs) && olangs[oi] != "" {
				lang = olangs[oi]
			}
			oi++
			if lang != "" {
				if i := strings.IndexByte(ln, '`'); i >= 0 {
					rest := ln[i:]
					if strings.HasPrefix(rest, "```") {
						b.WriteString("```" + lang)
						if tail := strings.TrimPrefix(rest, "```"); tail != "" && !strings.HasPrefix(strings.TrimSpace(tail), lang) {
							b.WriteString(tail)
						}
						b.WriteByte('\n')
						continue
					}
				}
			}
		}
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func fenceLangs(body string) []string {
	var langs []string
	fence := ""
	for _, ln := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trim, "```") {
			fence = "```"
			lang := strings.TrimSpace(strings.TrimPrefix(trim, "```"))
			langs = append(langs, lang)
		} else if strings.HasPrefix(trim, "~~~") {
			fence = "~~~"
			lang := strings.TrimSpace(strings.TrimPrefix(trim, "~~~"))
			langs = append(langs, lang)
		}
	}
	return langs
}
