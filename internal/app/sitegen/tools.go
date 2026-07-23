package sitegen

import "strings"

type toolDefinition struct {
	ID           string
	Template     string
	Script       string
	Icon         string
	Translations map[string]toolCopy
}

type toolCopy struct {
	Title         string
	Description   string
	InputLabel    string
	OutputLabel   string
	TreeLabel     string
	FormatLabel   string
	MinifyLabel   string
	EncodeLabel   string
	DecodeLabel   string
	CopyLabel     string
	ClearLabel    string
	Hint          string
	NeedInput     string
	InputTooLarge string
	Copied        string
	CopyFailed    string
	Cleared       string
	InvalidBase64 string
	InvalidJSON   string
	TreeTruncated string
}

type toolCardView struct {
	ID          string
	Title       string
	Description string
	URL         string
	Icon        string
}

var toolDefinitions = []toolDefinition{
	{
		ID:       "base64",
		Template: "tool-base64.html",
		Script:   "/assets/tool-base64.js",
		Icon:     "binary",
		Translations: map[string]toolCopy{
			"zh": {Title: "Base64 编解码", Description: "在浏览器本地进行 UTF-8 文本与 Base64 的互相转换。", InputLabel: "输入", OutputLabel: "结果", EncodeLabel: "编码", DecodeLabel: "解码", CopyLabel: "复制结果", ClearLabel: "清空", Hint: "内容不会发送到服务器。", NeedInput: "请输入内容。", InputTooLarge: "输入不能超过 1 MB。", Copied: "已复制结果。", CopyFailed: "无法复制，请手动选择结果。", Cleared: "已清空。", InvalidBase64: "请输入有效的 UTF-8 Base64 内容。"},
			"en": {Title: "Base64 encoder and decoder", Description: "Convert UTF-8 text and Base64 entirely in your browser.", InputLabel: "Input", OutputLabel: "Result", EncodeLabel: "Encode", DecodeLabel: "Decode", CopyLabel: "Copy result", ClearLabel: "Clear", Hint: "Your content is never sent to a server.", NeedInput: "Enter some content first.", InputTooLarge: "Input must be no larger than 1 MB.", Copied: "Result copied.", CopyFailed: "Could not copy. Select the result manually.", Cleared: "Cleared.", InvalidBase64: "Enter valid UTF-8 Base64 content."},
		},
	},
	{
		ID:       "json",
		Template: "tool-json.html",
		Script:   "/assets/tool-json.js",
		Icon:     "braces",
		Translations: map[string]toolCopy{
			"zh": {Title: "JSON 可视化", Description: "在浏览器本地格式化、压缩和查看 JSON 数据结构。", InputLabel: "JSON 输入", OutputLabel: "格式化结果", TreeLabel: "数据结构", FormatLabel: "格式化", MinifyLabel: "压缩", CopyLabel: "复制结果", ClearLabel: "清空", Hint: "最大支持 1 MB 输入，内容不会发送到服务器。", NeedInput: "请输入 JSON 内容。", InputTooLarge: "输入不能超过 1 MB。", Copied: "已复制结果。", CopyFailed: "无法复制，请手动选择结果。", Cleared: "已清空。", InvalidJSON: "JSON 格式无效。", TreeTruncated: "预览已截断，以保护浏览器性能。"},
			"en": {Title: "JSON visualizer", Description: "Format, minify, and inspect JSON data structures locally in your browser.", InputLabel: "JSON input", OutputLabel: "Formatted result", TreeLabel: "Data structure", FormatLabel: "Format", MinifyLabel: "Minify", CopyLabel: "Copy result", ClearLabel: "Clear", Hint: "Input is limited to 1 MB and never sent to a server.", NeedInput: "Enter JSON content first.", InputTooLarge: "Input must be no larger than 1 MB.", Copied: "Result copied.", CopyFailed: "Could not copy. Select the result manually.", Cleared: "Cleared.", InvalidJSON: "The JSON is invalid.", TreeTruncated: "Preview was truncated to protect browser performance."},
		},
	},
}

func staticTools(locale string) []toolCardView {
	tools := make([]toolCardView, 0, len(toolDefinitions))
	for _, definition := range toolDefinitions {
		copy := definition.copyFor(locale)
		tools = append(tools, toolCardView{ID: definition.ID, Title: copy.Title, Description: copy.Description, URL: toolRoute(locale, definition.ID), Icon: definition.Icon})
	}
	return tools
}

func (d toolDefinition) copyFor(locale string) toolCopy {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return d.Translations["zh"]
	}
	return d.Translations["en"]
}
