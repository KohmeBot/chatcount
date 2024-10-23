// Package chatcount 聊天时长统计
package chatcount

import (
	"fmt"
	"github.com/kohmebot/pkg/command"
	"github.com/kohmebot/pkg/version"
	"github.com/kohmebot/plugin"
	zero "github.com/wdvxdr1123/ZeroBot"
)

const (
	rankSize = 10
)

type PluginChatCount struct {
	env      plugin.Env
	ctdb     *chattimedb
	l        *leveler
	conf     Config
	filePath string
}

func NewPlugin() plugin.Plugin {
	return new(PluginChatCount)
}

func (p *PluginChatCount) Init(engine *zero.Engine, env plugin.Env) error {
	p.env = env
	err := env.GetConf(&p.conf)
	if err != nil {
		return err
	}
	p.filePath, err = env.FilePath()
	if err != nil {
		return err
	}
	db, err := env.GetDB()
	if err != nil {
		return err
	}
	p.ctdb, err = initialize(db)
	if err != nil {
		return err
	}

	p.SetOnMsg(engine)
	p.SetOnRankSearch(engine)
	p.SetOnTimeSearch(engine)

	return nil
}

func (p *PluginChatCount) Name() string {
	return "chatcount"
}

func (p *PluginChatCount) Description() string {
	return "统计水群时长"
}

func (p *PluginChatCount) Commands() fmt.Stringer {
	return command.NewCommands(
		command.NewCommand("查看当前水群情况", "查询水群"),
		command.NewCommand("查看当日水群排行", "水群排名"),
	)
}

func (p *PluginChatCount) Version() uint64 {
	return uint64(version.NewVersion(1, 0, 41))
}

func (p *PluginChatCount) OnBoot() {
	p.startRankSendTicker()
}
