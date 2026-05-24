package component

type Form struct {
	Fields map[string]string
}

func (f Form) Value(name string) string {
	if f.Fields == nil {
		return ""
	}
	return f.Fields[name]
}
