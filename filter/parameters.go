package query

import (
    "fmt"
)

type Parameters map[string]interface{}

func substituteParameters(node Node, params Parameters) (Node, error) {
    switch n := node.(type) {
    case *ParameterNode:
        if val, ok := params[n.Name]; ok {
            return &ValueNode{Value: val}, nil
        }
        return nil, fmt.Errorf("missing parameter: %s", n.Name)
    }
    return node, nil
}
