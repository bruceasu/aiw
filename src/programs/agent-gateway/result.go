package main

type RunResult struct {
	Started bool
	Texts []string
	ToolCall *FunctionCall
	Usage *Usage
	State string
	Code string
	HTTP int
}
