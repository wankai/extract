package extract

type combineItem struct {
	replace bool
	content string
}

type Combine []combineItem

func MakeCombine(s string) Combine {
	cb := make([]combineItem, 0)
	start := 0
	i := 0
	length := len(s)

	for i < length {
		if s[i] == '$' {
			if s[i+1] == '{' {
				j := i + 1
				for ; j < length; j++ {
					if s[j] == '}' {
						break
					}
				}
				if j == length {
					i++
				} else {
					normal := s[start:i]
					if len(normal) > 0 {
						cb = append(cb, combineItem{replace:false, content:normal})
					}
					cb = append(cb, combineItem{replace: true, content: s[i+2:j]})
					i = j + 1
					start = i
				}
			} else if s[i+1] >= '0' && s[i+1] <= '9' {
				normal := s[start:i]
				if len(normal) > 0 {
					cb = append(cb, combineItem{replace:false, content:normal})
				}
				cb = append(cb, combineItem{replace:true, content:s[i+1:i+2]})
				i = i + 2
				start = i
			} else {
				i++
			}
		} else {
			i++
		}
	}

	if start < length {
		normal := s[start:i]
		if len(normal) > 0 {
			cb = append(cb, combineItem{replace: false, content:normal})
		}
	}
	return cb
}

func (cb Combine) String() string {
	s := ""
	for _, p := range(cb) {
		content := p.content
		if p.replace {
			s += "${" + content + "}"
		} else {
			s += content
		}
	}
	return s
}

func (cb Combine) Exec(data map[string]string) string {
	s := ""
	for _, p := range(cb) {
		content := p.content
		if !p.replace {
			s += content
			continue
		}
		v, ok := data[content]
		if !ok {
			return ""
		}
		s += v
	}
	return s
}

func (cb Combine) ExecLoose(data map[string]string) string {
	s := ""
	for _, p := range(cb) {
		content := p.content
		if !p.replace {
			s += content
			continue
		}
		if v, ok := data[content]; ok {
			s += v
		}
	}
	return s
}
