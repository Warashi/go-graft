package astcow

import (
	"errors"
	"fmt"
	"go/ast"
)

var ErrUnsupportedNode = errors.New("astcow: unsupported node type")

// ClonePath applies one-node mutation with copy-on-write on the given path.
func ClonePath(pathOrig []ast.Node, nodeOrig ast.Node, nodeMut ast.Node) (*ast.File, map[ast.Node]ast.Node, error) {
	if len(pathOrig) == 0 {
		return nil, nil, errors.New("astcow: empty path")
	}
	if nodeOrig == nil || nodeMut == nil {
		return nil, nil, errors.New("astcow: nodeOrig and nodeMut must not be nil")
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
			return nil, nil, fmt.Errorf("astcow: failed to replace child %T in %T", childOrig, parentOrig)
		}
		childOrig = parentOrig
		childClone = parentClone
	}

	file, ok := childClone.(*ast.File)
	if !ok {
		return nil, nil, fmt.Errorf("astcow: root clone type %T, want *ast.File", childClone)
	}
	return file, cloneMap, nil
}
