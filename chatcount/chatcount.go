// Package chatcount 聊天时长统计
package chatcount

import (
	"github.com/kohmebot/pkg/command"
	"github.com/kohmebot/plugin/v2"
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
	"牛", "6", "一个",
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

func (p *PluginChatCount) GetGroupChatInfo(group int64, onlyToday bool) map[int64][2]int64 {
	if onlyToday {
		return p.ctdb.getChatInfo(group)
	}
	return p.ctdb.getTotalChatInfo(group)
}

func (p *PluginChatCount) OnInit(engine plugin.Engine, env plugin.Env) error {
	p.env = env
	err := env.GetConf(&p.conf)
	if err != nil {
		return err
	}
	p.filePath, err = env.FilePath()
	if err != nil {
		return err
	}
	initJieBaDict(p.filePath)
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

func (p *PluginChatCount) OnHelp(ctx *zero.Ctx) {
	help := command.HelpTemplate{
		PluginName: "chatcount",
		PluginDesc: "记录水群数据",
		Commands: []command.Command{
			{
				CMD:  "水群查询",
				Desc: "查看当前水群情况",
			},
			{
				CMD:  "水群排名",
				Desc: "查看当日水群排名",
			},
		},
	}

	ctx.Send(help.String())
}

func (p *PluginChatCount) Version() string {
	return "v1.1.1"
}

func (p *PluginChatCount) OnBoot() {
	var err error
	defer func() {
		if err != nil {
			p.env.UseBot(func(ctx *zero.Ctx) {
				p.env.Error(ctx, err)
			})

		}
	}()
	p.startRankSendTicker()
	err = p.ctdb.autoClear()

}

func initJieBaDict(rootPath string) {
	gojieba.DICT_DIR = filepath.Join(rootPath, "dict")
	gojieba.DICT_PATH = filepath.Join(gojieba.DICT_DIR, "jieba.dict.utf8")
	gojieba.HMM_PATH = filepath.Join(gojieba.DICT_DIR, "hmm_model.utf8")
	gojieba.USER_DICT_PATH = filepath.Join(gojieba.DICT_DIR, "user.dict.utf8")
	gojieba.IDF_PATH = filepath.Join(gojieba.DICT_DIR, "idf.utf8")
	gojieba.STOP_WORDS_PATH = filepath.Join(gojieba.DICT_DIR, "stop_words.utf8")
}
