package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type jsonFrame struct {
	kind       json.Delim
	path       string
	keys       map[string]struct{}
	expectKey  bool
	pendingKey string
	nextIndex  int
}

func detectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	stack := make([]jsonFrame, 0)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parse config JSON: %w", err)
		}

		switch value := tok.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				path := nextContainerPath(stack)
				frame := jsonFrame{kind: value, path: path}
				if value == '{' {
					frame.keys = make(map[string]struct{})
					frame.expectKey = true
				}
				stack = append(stack, frame)
			case '}', ']':
				if len(stack) == 0 {
					return fmt.Errorf("parse config JSON: unexpected closing delimiter %q", string(value))
				}
				last := stack[len(stack)-1]
				if (value == '}' && last.kind != '{') || (value == ']' && last.kind != '[') {
					return fmt.Errorf("parse config JSON: mismatched closing delimiter %q", string(value))
				}
				stack = stack[:len(stack)-1]
				consumeValue(&stack)
			}
		case string:
			if len(stack) > 0 && stack[len(stack)-1].kind == '{' && stack[len(stack)-1].expectKey {
				frame := &stack[len(stack)-1]
				if _, exists := frame.keys[value]; exists {
					return fmt.Errorf("duplicate key %q at %s", value, frame.path)
				}
				frame.keys[value] = struct{}{}
				frame.pendingKey = value
				frame.expectKey = false
			} else {
				consumeValue(&stack)
			}
		default:
			consumeValue(&stack)
		}
	}

	if len(stack) != 0 {
		return fmt.Errorf("parse config JSON: unexpected end of document")
	}
	return nil
}

func nextContainerPath(stack []jsonFrame) string {
	if len(stack) == 0 {
		return "$"
	}
	parent := stack[len(stack)-1]
	if parent.kind == '{' && !parent.expectKey {
		if parent.path == "$" {
			return "$." + parent.pendingKey
		}
		return parent.path + "." + parent.pendingKey
	}
	if parent.kind == '[' {
		return fmt.Sprintf("%s[%d]", parent.path, parent.nextIndex)
	}
	return "$"
}

func consumeValue(stack *[]jsonFrame) {
	if len(*stack) == 0 {
		return
	}
	frame := &(*stack)[len(*stack)-1]
	if frame.kind == '{' && !frame.expectKey {
		frame.pendingKey = ""
		frame.expectKey = true
		return
	}
	if frame.kind == '[' {
		frame.nextIndex++
	}
}
