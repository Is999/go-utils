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

// sourceMulti 在多错误场景下查找源错误。
// 返回第一个非 nil 分支的最底层错误。
//
// 参数说明：
//   - err：Join 多错误
//
// 返回值：第一个分支的源错误
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
