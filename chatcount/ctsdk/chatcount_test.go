package ctsdk

import (
	"github.com/kohmebot/chatcount/chatcount"
	"github.com/kohmebot/plugin/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockEnv struct {
	plugin.Env
}

func (e *mockEnv) GetPlugin(name string) (plugin.Plugin, bool) {
	return &chatcount.PluginChatCount{}, true
}

func TestCtInvoker_GetGroupChatInfo(t *testing.T) {
	i, err := NewCtInvoker(&mockEnv{})
	assert.NoError(t, err)
	i.GetGroupChatInfo(123, true)
}
