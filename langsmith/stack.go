package langsmith

type stack []*RunTree

func (s stack) Push(v *RunTree) stack {
	return append(s, v)
}

func (s stack) Pop() (stack, *RunTree) {
	if len(s) == 0 {
		panic("run tree stack is empty")
	}

	l := len(s)
	return s[:l-1], s[l-1]
}
