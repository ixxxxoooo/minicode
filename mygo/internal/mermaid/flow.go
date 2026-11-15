package mermaid

func renderFlow(src string) (Scene, error) {
	flow, err := parseFlow(src)
	if err != nil {
		return Scene{}, err
	}
	if len(flow.Nodes) == 0 {
		return Scene{}, RenderError{Kind: ErrEmpty}
	}
	return drawFlow(flow), nil
}
