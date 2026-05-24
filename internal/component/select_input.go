package component

type SelectInput struct {
	Options []string
	Index   int
}

func (s SelectInput) Selected() string {
	if len(s.Options) == 0 || s.Index < 0 || s.Index >= len(s.Options) {
		return ""
	}
	return s.Options[s.Index]
}
