// Package chatcount 聊天时长统计
package chatcount

import (
	"fmt"
	"github.com/kohmebot/pkg/command"
	"github.com/kohmebot/pkg/version"
	"github.com/kohmebot/plugin"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/yanyiwu/gojieba"
	"os"
	"path/filepath"
	"strings"
)

const (
	rankSize = 10
)

var filter = []string{
	"牛", "6",
}

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
	var stopWordsPath string
	if len(p.conf.StopWordFile) <= 0 {
		stopWordsPath = gojieba.STOP_WORDS_PATH
	} else {
		stopWordsPath = filepath.Join(p.filePath, p.conf.StopWordFile)
	}
	buf, err := os.ReadFile(stopWordsPath)
	if err != nil {
		return err
	}
	p.ctdb, err = initialize(db, strings.Fields(string(buf)))
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
		command.NewCommand("查看当前水群情况", "水群查询"),
		command.NewCommand("查看当日水群排行", "水群排名"),
	)
}

func (p *PluginChatCount) Version() uint64 {
	return uint64(version.NewVersion(1, 0, 50))
}

func (p *PluginChatCount) OnBoot() {
	var err error
	defer func() {
		if err != nil {
			for ctx := range p.env.RangeBot {
				p.env.Error(ctx, err)
			}
		}
	}()
	p.startRankSendTicker()
	err = p.ctdb.autoClear()

}
