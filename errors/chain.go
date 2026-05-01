package errors

// ============================ 错误链遍历函数 ============================

// Cause 返回错误链最底层的源错误。
// 是 Source 的别名，功能完全相同。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：最底层的源错误，nil 错误返回 nil
func Cause(err error) error { return Source(err) }

// Root 返回错误链最底层的源错误。
// 是 Source 的别名，功能完全相同。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：最底层的源错误，nil 错误返回 nil
func Root(err error) error { return Source(err) }

// HasStack 检查错误链路中是否已经存在追踪栈。
// 用于判断是否需要重复采集栈信息，避免 Wrap 时重复调用 runtime.Callers。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：true 表示错误链中存在带栈追踪的错误
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

// Source 返回错误链最底层的源错误。
// 沿错误链向下遍历，找到第一个不包含 unwrap 的错误节点。
// 对 Join 多错误场景，返回第一个非 nil 分支的源错误。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：最底层的源错误
func Source(err error) error {
	if err == nil {
		return nil
	}

	last := err
	for current, depth := err, 0; current != nil && depth < maxChainDepth; depth++ {
		last = current
		next, children := unwrapNode(current)
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

// Sources 返回错误链中所有最终源错误。
// 对 Join 多错误场景，返回所有分支的最底层错误。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：所有源错误的切片，按从左到右的顺序排列
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
		next, children := unwrapNode(current)
		switch {
		case len(children) > 0:
			before := len(stack)
			pushChildren(&stack, children)
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

// Chain 返回错误链中的所有节点。
// 对 Join 多错误场景，按从左到右的深度优先顺序返回所有节点。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：错误链中所有节点的切片
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
		next, children := unwrapNode(current)
		switch {
		case len(children) > 0:
			pushChildren(&stack, children)
		case next != nil:
			stack = append(stack, next)
		}
	}
	return chain
}

// ============================ 内部辅助函数 ============================

// popError 从栈顶弹出一个错误。
// 使用后进先出策略，模拟递归遍历。
//
// 参数说明：
//   - stack：指向错误切片的指针
//
// 返回值：栈顶的错误对象
func popError(stack *[]error) error {
	last := len(*stack) - 1
	current := (*stack)[last]
	*stack = (*stack)[:last]
	return current
}

// pushChildren 将子错误入栈。
// 为保证深度优先遍历的正确顺序，从后向前遍历子错误切片入栈。
//
// 参数说明：
//   - stack：指向错误切片的指针
//   - children：子错误列表
func pushChildren(stack *[]error, children []error) {
	for i := len(children) - 1; i >= 0; i-- {
		if children[i] != nil {
			*stack = append(*stack, children[i])
		}
	}
}

// compactErrors 压缩错误列表，过滤 nil 错误。
// 使用原地压缩算法，无 nil 时直接返回原切片，避免堆分配。
//
// 参数说明：
//   - errs：错误列表
//
// 返回值：过滤 nil 后的错误列表
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

// hasStackMulti 检查多错误列表中是否存在带栈追踪的错误。
// 用于检测 errors.Join 包裹的场景。
//
// 参数说明：
//   - children：子错误列表
//
// 返回值：true 表示存在带栈追踪的错误
func hasStackMulti(children []error) bool {
	var stackBuf [8]error
	stack := stackBuf[:0]
	pushChildren(&stack, children)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
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
			next, children := unwrapNode(current)
			switch {
			case len(children) > 0:
				pushChildren(&stack, children)
			case next != nil:
				stack = append(stack, next)
			}
		}
	}
	return false
}

// sourceMulti 在多错误场景下查找源错误。
// 返回第一个非 nil 分支的最底层错误。
//
// 参数说明：
//   - err：Join 多错误
//
// 返回值：第一个分支的源错误
func sourceMulti(err error, children []error) error {
	var stackBuf [8]error
	stack := stackBuf[:0]
	pushChildren(&stack, children)
	if len(stack) == 0 {
		return err
	}
	var last error
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		current := popError(&stack)
		if current == nil {
			continue
		}
		last = current
		next, children := unwrapNode(current)
		switch {
		case len(children) > 0:
			pushChildren(&stack, children)
		case next != nil:
			stack = append(stack, next)
		default:
			return current
		}
	}
	return last
}

// unwrapNode 解析单分支或多分支错误节点。
//
// 返回值：
//   - error：单分支下一个错误节点。
//   - []error：errors.Join 等多分支错误节点。
func unwrapNode(err error) (error, []error) {
	switch e := err.(type) {
	case *stackError:
		return e.err, nil
	case *messageError:
		return e.err, nil
	case *codeError:
		return e.err, nil
	case *contextError:
		return e.err, nil
	case multiUnwrapper:
		return nil, e.Unwrap()
	case unwrapper:
		return e.Unwrap(), nil
	default:
		return nil, nil
	}
}
