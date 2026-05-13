// JSON serialization for KDL documents, nodes, and values, in the format that
// kdl-test expects. Not intended for general use, just for testing.

package kdl

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// MarshalJSON implements [json.Marshaler] for Document, encoding the document
// as a JSON array of nodes, matching the expectations of kdl-test. The JSON
// format is not intended to be stable or for general use.
func (d Document) MarshalJSON() ([]byte, error) {
	nodes := d.Nodes
	if nodes == nil {
		nodes = []*Node{}
	}
	return json.Marshal(nodes)
}

// MarshalJSON implements [json.Marshaler] for Node, encoding the node as a JSON
// object with "name", "type", "args", "props", and "children" fields, matching
// the expectations of kdl-test. The JSON format is not intended to be stable or
// for general use.
func (n *Node) MarshalJSON() ([]byte, error) {
	j := map[string]any{
		"name":     n.name,
		"type":     nil,
		"args":     []Value{},
		"props":    n.props,
		"children": n.children,
	}
	if n.typeValid {
		j["type"] = &n.typ
	}
	if len(n.args) > 0 {
		j["args"] = n.args
	}
	return json.Marshal(j)
}

// MarshalJSON implements [json.Marshaler] for Value, encoding the value as a
// JSON object with "type" and "value" fields, matching the expectations of
// kdl-test. The JSON format is not intended to be stable or for general use.
func (v Value) MarshalJSON() ([]byte, error) {
	var val map[string]string
	switch v.Kind() {
	case String:
		val = map[string]string{"type": "string", "value": v.String()}
	case Bool:
		str := "false"
		if v.Bool() {
			str = "true"
		}
		val = map[string]string{"type": "boolean", "value": str}
	case Null:
		val = map[string]string{"type": "null"}
	case Int:
		val = map[string]string{"type": "number", "value": fmt.Sprintf("%d.0", v.Int())}
	case Float:
		str := fmt.Sprintf("%f", v.Float())
		if math.IsInf(v.Float(), 1) {
			str = "inf"
		} else if math.IsInf(v.Float(), -1) {
			str = "-inf"
		} else if math.IsNaN(v.Float()) {
			str = "nan"
		} else {
			// remove extra trailing zeros (e.g. "1.000" -> "1.0")
			str = strings.TrimRight(str, "0")
			if str[len(str)-1] == '.' {
				str += "0"
			}
		}
		val = map[string]string{"type": "number", "value": str}
	case BigInt:
		val = map[string]string{"type": "number", "value": fmt.Sprintf("%d.0", v.BigInt())}
	case BigFloat:
		str := v.BigFloat().Text('f', -1)
		// add ".0" if needed
		if !strings.ContainsAny(str, ".") {
			str += ".0"
		}
		val = map[string]string{"type": "number", "value": str}
	default:
		return nil, nil
	}

	var typ *string
	if v.typeValid {
		typ = &v.typ
	}

	return json.Marshal(map[string]any{
		"type":  typ,
		"value": val,
	})
}
