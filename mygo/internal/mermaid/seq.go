package mermaid

func renderSeq(src string) (Scene, error) {
	seq, err := parseSeq(src)
	if err != nil {
		return Scene{}, err
	}
	if len(seq.Parts) > MaxNodes {
		return Scene{}, RenderError{Kind: ErrTooLarge, N: MaxNodes}
	}
	return drawSeq(seq), nil
}
