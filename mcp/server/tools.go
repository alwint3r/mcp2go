package server

import (
	"context"
	"errors"
)

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Annotations interface{}            `json:"annotations,omitempty"`
}

type ToolCallContent struct {
	Type     string  `json:"type"`
	Text     *string `json:"text,omitempty"`
	Data     *string `json:"data,omitempty"` // base64-encoded string
	MimeType *string `json:"mimeType,omitempty"`
}

type ToolResult struct {
	Content []ToolCallContent
	IsError bool
}

type ToolCallback func(context.Context, string, map[string]interface{}) ToolResult
type ToolCallbacksMap map[string]ToolCallback

type ToolManager struct {
	tools         []Tool
	toolCallbacks ToolCallbacksMap
}

func (t *ToolManager) findTool(name string) (*Tool, *ToolCallback, error) {
	var tool *Tool
	for _, toolDef := range t.tools {
		if toolDef.Name == name {
			tool = &toolDef
		}
	}

	if tool == nil {
		return nil, nil, errors.New("tool not found")
	}

	callback, exist := t.toolCallbacks[name]
	if !exist {
		return nil, nil, errors.New("can't execute tool, no callback")
	}

	return tool, &callback, nil
}

func (t *ToolManager) AddTool(definition Tool, callback ToolCallback) {
	t.tools = append(t.tools, definition)
	t.toolCallbacks[definition.Name] = callback
}

func (t *ToolManager) CallTool(ctx context.Context, name string, arguments map[string]interface{}) ToolResult {
	toolDef, toolCallback, err := t.findTool(name)
	if err != nil {
		errString := err.Error()
		return ToolResult{
			Content: []ToolCallContent{
				{
					Type: "text",
					Text: &errString,
				},
			},
			IsError: true,
		}
	}

	callback := *toolCallback
	result := callback(ctx, toolDef.Name, arguments)
	return result
}

func (t *ToolManager) ListAllTools() *[]Tool {
	return &t.tools
}

func NewToolManager() ToolManager {
	return ToolManager{
		tools:         make([]Tool, 0),
		toolCallbacks: make(ToolCallbacksMap),
	}
}
