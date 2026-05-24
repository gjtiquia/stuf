package component

type TextInput struct {
	Label string
	Value string
}

func (i TextInput) View() string {
	return i.Label + ": " + i.Value
}
