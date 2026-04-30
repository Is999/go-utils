package errors

// Cause 返回错误链最底层的源错误，是 Source 的别名。
func Cause(err error) error { return Source(err) }

// Root 返回错误链最底层的源错误，是 Source 的别名。
func Root(err error) error { return Source(err) }

// HasStack 检查错误链路中是否已经存在追踪栈。
func HasStack(err error) bool {
	if err == nil {
		return false
	}
	for current := err; current != nil; {
		if _, ok := current.(*stackError); ok {
			return true
		}
		if mu, ok := current.(multiUnwrapper); ok {
			return hasStackMulti(mu.Unwrap())
		}
		next, ok := current.(unwrapper)
		if !ok {
			return false
		}
		current = next.Unwrap()
	}
	return false
}

// Source 返回错误链最底层的源错误。
// 对 Join 场景，返回第一个非 nil 分支的源错误。
func Source(err error) error {
	if err == nil {
		return nil
	}

	for current := err; current != nil; {
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			return sourceMulti(current)
		case info.next != nil:
			current = info.next
		default:
			return current
		}
	}

	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	var last error
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		last = current
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			pushChildren(&stack, info.children)
		case info.next != nil:
			stack = append(stack, info.next)
		default:
			return current
		}
	}
	return last
}

// Sources 返回错误链中所有最终源错误。
func Sources(err error) []error {
	if err == nil {
		return nil
	}
	sources := make([]error, 0, 8)
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			before := len(stack)
			pushChildren(&stack, info.children)
			if len(stack) == before {
				sources = append(sources, current)
			}
		case info.next != nil:
			stack = append(stack, info.next)
		default:
			sources = append(sources, current)
		}
	}
	return sources
}

// Chain 返回错误链中的所有节点。
// 对 Join 场景，按从左到右的深度优先顺序返回。
func Chain(err error) []error {
	if err == nil {
		return nil
	}
	chain := make([]error, 0, 16)
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		chain = append(chain, current)
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			pushChildren(&stack, info.children)
		case info.next != nil:
			stack = append(stack, info.next)
		}
	}
	return chain
}

func popError(stack *[]error) error {
	last := len(*stack) - 1
	current := (*stack)[last]
	*stack = (*stack)[:last]
	return current
}

func pushChildren(stack *[]error, children []error) {
	for i := len(children) - 1; i >= 0; i-- {
		if children[i] != nil {
			*stack = append(*stack, children[i])
		}
	}
}

func compactErrors(errs []error) []error {
	if len(errs) == 0 {
		return nil
	}
	firstNil := -1
	for i, err := range errs {
		if err == nil {
			if firstNil < 0 {
				firstNil = i
			}
		} else if firstNil >= 0 {
			errs[firstNil] = err
			firstNil++
		}
	}
	if firstNil < 0 {
		return errs
	}
	return errs[:firstNil]
}

func hasStackMulti(children []error) bool {
	var stackBuf [8]error
	stack := stackBuf[:0]
	pushChildren(&stack, children)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		if _, ok := current.(*stackError); ok {
			return true
		}
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			pushChildren(&stack, info.children)
		case info.next != nil:
			stack = append(stack, info.next)
		}
	}
	return false
}

func sourceMulti(err error) error {
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	var last error
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		last = current
		info := inspectError(current)
		switch {
		case len(info.children) > 0:
			pushChildren(&stack, info.children)
		case info.next != nil:
			stack = append(stack, info.next)
		default:
			return current
		}
	}
	return last
}
