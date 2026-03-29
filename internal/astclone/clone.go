package astclone

import (
	"errors"
	"fmt"
	"go/ast"
)

var ErrUnsupportedNode = errors.New("astclone: unsupported node type")

// ShallowCopyNode returns a one-level copy for supported node types.
func ShallowCopyNode(n ast.Node) ast.Node {
	return shallowCopy(n)
}

// DeepCopyNode returns a deep copy of n and clone-to-original mapping.
func DeepCopyNode(n ast.Node) (ast.Node, map[ast.Node]ast.Node, error) {
	if n == nil {
		return nil, nil, errors.New("astclone: node must not be nil")
	}

	cloneToOriginal := make(map[ast.Node]ast.Node)
	originalToClone := make(map[ast.Node]ast.Node)
	var rootClone ast.Node
	var walkErr error

	ast.PreorderStack(n, nil, func(current ast.Node, stack []ast.Node) bool {
		if walkErr != nil {
			return false
		}

		cloned, seen := originalToClone[current]
		if !seen {
			cloned = shallowCopy(current)
			if cloned == nil {
				walkErr = fmt.Errorf("%w: %T", ErrUnsupportedNode, current)
				return false
			}
			originalToClone[current] = cloned
			cloneToOriginal[cloned] = current
		}

		if len(stack) == 0 {
			rootClone = cloned
			return !seen
		}

		parentOrig := stack[len(stack)-1]
		parentClone, ok := originalToClone[parentOrig]
		if !ok {
			walkErr = fmt.Errorf("astclone: missing parent clone for %T", parentOrig)
			return false
		}
		if !replaceChild(parentClone, current, cloned) {
			walkErr = fmt.Errorf("astclone: failed to replace child %T in %T", current, parentOrig)
			return false
		}
		return !seen
	})

	if walkErr != nil {
		return nil, nil, walkErr
	}
	if rootClone == nil {
		return nil, nil, errors.New("astclone: failed to clone root")
	}
	return rootClone, cloneToOriginal, nil
}

// ClonePath applies one-node mutation with copy-on-write on the given path.
func ClonePath(pathOrig []ast.Node, nodeOrig ast.Node, nodeMut ast.Node) (*ast.File, map[ast.Node]ast.Node, error) {
	if len(pathOrig) == 0 {
		return nil, nil, errors.New("astclone: empty path")
	}
	if nodeOrig == nil || nodeMut == nil {
		return nil, nil, errors.New("astclone: nodeOrig and nodeMut must not be nil")
	}

	cloneMap := map[ast.Node]ast.Node{
		nodeMut: nodeOrig,
	}

	childOrig := nodeOrig
	childClone := nodeMut
	for i := len(pathOrig) - 2; i >= 0; i-- {
		parentOrig := pathOrig[i]
		parentClone := shallowCopy(parentOrig)
		if parentClone == nil {
			return nil, nil, fmt.Errorf("%w: %T", ErrUnsupportedNode, parentOrig)
		}
		cloneMap[parentClone] = parentOrig
		if !replaceChild(parentClone, childOrig, childClone) {
			return nil, nil, fmt.Errorf("astclone: failed to replace child %T in %T", childOrig, parentOrig)
		}
		childOrig = parentOrig
		childClone = parentClone
	}

	file, ok := childClone.(*ast.File)
	if !ok {
		return nil, nil, fmt.Errorf("astclone: root clone type %T, want *ast.File", childClone)
	}
	return file, cloneMap, nil
}
