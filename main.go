package main

import (
	"github.com/kohmebot/chatcount/chatcount"
	"github.com/kohmebot/plugin"
)

func NewPlugin() plugin.Plugin {
	return chatcount.NewPlugin()
}
