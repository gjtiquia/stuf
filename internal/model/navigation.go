package model

type navFrame struct {
	Path string
	Menu int
}

type NavigationStack struct {
	frames []navFrame
}

func NewNavigationStack() NavigationStack {
	return NavigationStack{frames: []navFrame{{Path: "/", Menu: 0}}}
}

func (s NavigationStack) Current() navFrame {
	if len(s.frames) == 0 {
		return navFrame{Path: "/", Menu: 0}
	}
	return s.frames[len(s.frames)-1]
}

func (s NavigationStack) Len() int {
	return len(s.frames)
}

func (s *NavigationStack) setCurrentMenu(menu int) {
	if len(s.frames) == 0 {
		return
	}
	s.frames[len(s.frames)-1].Menu = menu
}

func (s *NavigationStack) Push(path string, menu int) {
	s.frames = append(s.frames, navFrame{Path: path, Menu: menu})
}

func (s *NavigationStack) Replace(path string, menu int) {
	if len(s.frames) == 0 {
		s.frames = []navFrame{{Path: path, Menu: menu}}
		return
	}
	s.frames[len(s.frames)-1] = navFrame{Path: path, Menu: menu}
}

func (s *NavigationStack) Pop() bool {
	if len(s.frames) <= 1 {
		return false
	}
	s.frames = s.frames[:len(s.frames)-1]
	return true
}

func (s *NavigationStack) Reset() {
	s.frames = []navFrame{{Path: "/", Menu: 0}}
}

func (s *NavigationStack) SetBelowTop(path string, menu int) bool {
	if len(s.frames) < 2 {
		return false
	}
	s.frames[len(s.frames)-2] = navFrame{Path: path, Menu: menu}
	return true
}
