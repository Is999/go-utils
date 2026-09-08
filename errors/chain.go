package errors

import "slices"

// Cause 是 Source 的别名；多个分支时仅返回首个源错误。
func Cause(err error) error { return Source(err) }

// Root 是 Source 的别名；需要全部分支时使用 Sources。
func Root(err error) error { return Source(err) }

// HasStack 只识别本包采集的栈，Join 任一分支命中即返回 true。
func HasStack(err error) bool {
	if err == nil {
		return false
	}
	for current, depth := err, 0; current != nil && depth < maxChainDepth; depth++ {
		switch e := current.(type) {
		case *stackError:
			return true
		case *messageError:
			current = e.err
		case *codeError:
			current = e.err
		case *contextError:
			current = e.err
		case multiUnwrapper:
			return hasStackMulti(e.Unwrap())
		case unwrapper:
			current = e.Unwrap()
		default:
			return false
		}
	}
	return false
}

// Source 从外向内查找首个源错误，Join 按从左到右的深度优先顺序展开。
// 达到遍历上限时返回最后访问的节点，不保证已到达叶子。
func Source(err error) error {
	if err == nil {
		return nil
	}

	last := err
	for current, depth := err, 0; current != nil && depth < maxChainDepth; depth++ {
		last = current
		children, next := unwrapNode(current)
		if len(children) > 0 {
			return sourceMulti(current, children)
		}
		if next == nil {
			return current
		}
		current = next
	}

	return last
}

// Sources 按深度优先顺序收集叶子错误，不去重；nil 返回 nil，达到遍历上限后停止。
func Sources(err error) []error {
	if err == nil {
		return nil
	}
	sources := make([]error, 0, 8)
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current == nil {
			continue
		}
		children, next := unwrapNode(current)
		switch {
		case len(children) > 0:
			before := len(stack)
			stack = pushChildren(stack, children)
			if len(stack) == before {
				sources = append(sources, current)
			}
		case next != nil:
			stack = append(stack, next)
		default:
			sources = append(sources, current)
		}
	}
	return sources
}

// Chain 按从左到右的深度优先顺序收集节点，不去重；nil 返回 nil，最多访问 1024 个节点。
func Chain(err error) []error {
	if err == nil {
		return nil
	}
	chain := make([]error, 0, 16)
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current == nil {
			continue
		}
		chain = append(chain, current)
		children, next := unwrapNode(current)
		switch {
		case len(children) > 0:
			stack = pushChildren(stack, children)
		case next != nil:
			stack = append(stack, next)
		}
	}
	return chain
}

// pushChildren 逆序压入非 nil 子节点，使出栈顺序与 Join 输入顺序一致。
func pushChildren(stack, children []error) []error {
	for i := len(children) - 1; i >= 0; i-- {
		if children[i] != nil {
			stack = append(stack, children[i])
		}
	}
	return stack
}

// compactErrors 忽略 nil 子节点，保留原顺序。
func compactErrors(errs []error) []error {
	if len(errs) == 0 {
		return nil
	}
	if slices.Contains(errs, nil) {
		// 子切片归原错误所有，仅需过滤时复制。
		return slices.DeleteFunc(slices.Clone(errs), func(err error) bool { return err == nil })
	}
	return errs
}

// hasStackMulti 使用显式栈展开分支，单链路径无需为此维护待遍历列表。
func hasStackMulti(children []error) bool {
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = pushChildren(stack, children)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current == nil {
			continue
		}
		switch e := current.(type) {
		case *stackError:
			return true
		case *messageError:
			stack = append(stack, e.err)
		case *codeError:
			stack = append(stack, e.err)
		case *contextError:
			stack = append(stack, e.err)
		default:
			children, next := unwrapNode(current)
			switch {
			case len(children) > 0:
				stack = pushChildren(stack, children)
			case next != nil:
				stack = append(stack, next)
			}
		}
	}
	return false
}

// sourceMulti 查找首个分支的叶子；没有非 nil 子节点时保留传入的错误。
func sourceMulti(err error, children []error) error {
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = pushChildren(stack, children)
	if len(stack) == 0 {
		return err
	}
	var last error
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		lastIndex := len(stack) - 1
		current := stack[lastIndex]
		stack = stack[:lastIndex]
		if current == nil {
			continue
		}
		last = current
		children, next := unwrapNode(current)
		switch {
		case len(children) > 0:
			stack = pushChildren(stack, children)
		case next != nil:
			stack = append(stack, next)
		default:
			return current
		}
	}
	return last
}

// unwrapNode 只展开一层，children 与 next 不同时返回；子切片归原错误所有。
func unwrapNode(err error) ([]error, error) {
	switch e := err.(type) {
	case *stackError:
		return nil, e.err
	case *messageError:
		return nil, e.err
	case *codeError:
		return nil, e.err
	case *contextError:
		return nil, e.err
	case multiUnwrapper:
		return e.Unwrap(), nil
	case unwrapper:
		return nil, e.Unwrap()
	default:
		return nil, nil
	}
}
