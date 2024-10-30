package chatcount

import (
	"errors"
	"fmt"
	"github.com/FloatTech/imgfactory"
	"github.com/FloatTech/rendercard"
	"github.com/fumiama/cron"
	"github.com/kohmebot/pkg/chain"
	"github.com/kohmebot/pkg/gopool"
	"github.com/sirupsen/logrus"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"image"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var noDataError = errors.New("没有水群数据")

func (p *PluginChatCount) SetOnMsg(engine *zero.Engine) {
	engine.OnMessage(p.env.Groups().Rule()).
		Handle(func(ctx *zero.Ctx) {
			gid, uid := ctx.Event.GroupID, ctx.Event.UserID
			p.ctdb.updateChatTime(gid, uid)
			msgs := make([]string, 0, 1)
			for _, segment := range ctx.Event.Message {
				if segment.Type == "text" {
					msgs = append(msgs, segment.Data["text"])
				}
			}
			p.ctdb.updateChatWord(gid, uid, msgs)
			//remindTime, remindFlag := p.ctdb.updateChatTime(ctx.Event.GroupID, ctx.Event.UserID)
			//if remindFlag {
			//	ctx.SendChain(message.At(ctx.Event.UserID), message.Text(fmt.Sprintf("BOT提醒：你今天已经水群%d分钟了！", remindTime)))
			//}
		})
}

func (p *PluginChatCount) SetOnTimeSearch(engine *zero.Engine) {
	engine.OnCommand("水群查询", p.env.Groups().Rule()).SetBlock(true).Handle(func(ctx *zero.Ctx) {
		name := ctx.NickName()
		todayTime, todayMessage, totalTime, totalMessage := p.ctdb.getChatTime(ctx.Event.GroupID, ctx.Event.UserID)
		ctx.SendChain(message.Reply(ctx.Event.MessageID), message.Text(fmt.Sprintf("%s今天水了%d分%d秒，发了%d条消息；总计水了%d分%d秒，发了%d条消息。", name, todayTime/60, todayTime%60, todayMessage, totalTime/60, totalTime%60, totalMessage)))
	})
}

func (p *PluginChatCount) SetOnRankSearch(engine *zero.Engine) {
	engine.OnCommand("水群排名", p.env.Groups().Rule()).SetBlock(true).
		Handle(func(ctx *zero.Ctx) {
			sendimg, err := p.getRankImage(ctx, ctx.Event.GroupID, p.conf.RankTitleTrigger)
			if err != nil {
				p.env.Error(ctx, err)
				return
			}
			if id := ctx.SendChain(message.ImageBytes(sendimg)); id.ID() == 0 {
				p.env.Error(ctx, fmt.Errorf("send image error"))
			}
		})
}

func (p *PluginChatCount) getRankImage(ctx *zero.Ctx, group int64, rankTitle string) ([]byte, error) {
	chatTimeList := p.ctdb.getChatRank(group)
	if len(chatTimeList) == 0 {
		return nil, noDataError
	}
	if len(chatTimeList) >= rankSize {
		// 超过rankSize，重新cut一下
		chatTimeList = chatTimeList[:rankSize]
	}
	rankinfo := make([]*rendercard.RankInfo, len(chatTimeList))
	wg := sync.WaitGroup{}
	for i := 0; i < len(chatTimeList); i++ {
		wg.Add(1)
		gopool.Go(func() {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("https://q4.qlogo.cn/g?b=qq&nk=%d&s=%d", chatTimeList[i].UserID, p.conf.AvatarSizeToParam()))
			if err != nil {
				return
			}
			defer resp.Body.Close()
			img, _, err := image.Decode(resp.Body)
			if err != nil {
				return
			}
			name := ctx.GetGroupMemberInfo(group, chatTimeList[i].UserID, false).Get("card").String()
			if name == "" {
				name = ctx.GetStrangerInfo(chatTimeList[i].UserID, false).Get("nickname").String()
			}

			rankinfo[i] = &rendercard.RankInfo{
				TopLeftText:    name,
				BottomLeftText: fmt.Sprintf("消息数: %d 条", chatTimeList[i].TodayMessage),
				RightText:      strconv.FormatInt(chatTimeList[i].TodayTime/60, 10) + "分" + strconv.FormatInt(chatTimeList[i].TodayTime%60, 10) + "秒",
				Avatar:         img,
			}
		})
	}
	wg.Wait()
	fontbyte, err := p.getFontData()
	if err != nil {
		return nil, err
	}
	img, err := rendercard.DrawRankingCard(fontbyte, rankTitle, rankinfo)
	if err != nil {
		return nil, err
	}
	return imgfactory.ToBytes(img)
}

func (p *PluginChatCount) getWordRankImage(ctx *zero.Ctx, group int64, rankTitle string) ([]byte, error) {
	chatTimeList := p.ctdb.getChatRank(group)
	if len(chatTimeList) == 0 {
		return nil, noDataError
	}
	if len(chatTimeList) >= rankSize {
		// 超过rankSize，重新cut一下
		chatTimeList = chatTimeList[:rankSize]
	}
	rankinfo := make([]*rendercard.RankInfo, len(chatTimeList))
	wg := sync.WaitGroup{}
	for i := 0; i < len(chatTimeList); i++ {
		wg.Add(1)
		gopool.Go(func() {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("https://q4.qlogo.cn/g?b=qq&nk=%d&s=%d", chatTimeList[i].UserID, p.conf.AvatarSizeToParam()))
			if err != nil {
				return
			}
			defer resp.Body.Close()
			img, _, err := image.Decode(resp.Body)
			if err != nil {
				return
			}
			name := ctx.GetGroupMemberInfo(group, chatTimeList[i].UserID, false).Get("card").String()
			if name == "" {
				name = ctx.GetStrangerInfo(chatTimeList[i].UserID, false).Get("nickname").String()
			}
			hotWords := chatTimeList[i].HotWords
			if len(hotWords) <= 0 {
				hotWords = []Word{
					{
						"无", 0,
					},
				}
			}
			var builder strings.Builder
			for _, word := range hotWords[1:] {
				builder.WriteString(fmt.Sprintf("%s(%d) ", word.Word, word.Count))
			}

			rankinfo[i] = &rendercard.RankInfo{
				TopLeftText:    name,
				BottomLeftText: builder.String(),
				RightText:      fmt.Sprintf("%s(%d)", hotWords[0].Word, hotWords[0].Count),
				Avatar:         img,
			}
		})
	}
	wg.Wait()
	fontbyte, err := p.getFontData()
	if err != nil {
		return nil, err
	}
	img, err := rendercard.DrawRankingCard(fontbyte, rankTitle, rankinfo)
	if err != nil {
		return nil, err
	}
	return imgfactory.ToBytes(img)
}

func (p *PluginChatCount) startRankSendTicker() {
	if len(p.conf.SendRankCron) <= 0 {
		return
	}
	c := cron.New()
	var id cron.EntryID
	id, err := c.AddFunc(p.conf.SendRankCron, func() {
		for ctx := range p.env.RangeBot {
			for group := range p.env.Groups().RangeGroup {
				rImgdata, err := p.getRankImage(ctx, group, p.conf.RankTitleTicker)
				time.Sleep(5 * time.Second)
				wImgdata, err := p.getWordRankImage(ctx, group, p.conf.WordRankTitleTicker)
				if err == nil {
					gopool.Go(func() {
						var msgChain chain.MessageChain
						if len(p.conf.MsgWithTicker) > 0 {
							msgChain.Line(message.Text(p.conf.MsgWithTicker))
						}
						msgChain.Join(message.ImageBytes(rImgdata))
						msgChain.Join(message.ImageBytes(wImgdata))
						ctx.SendGroupMessage(group, msgChain)
					})
				} else {
					if !errors.Is(err, noDataError) {
						p.env.Error(ctx, err)
					}
				}

			}
		}
		logrus.Infof("Next 将在 %s 发送Rank", c.Entry(id).Next)
	})
	if err != nil {
		logrus.Errorf("开启定时发送失败 %s", err)
		return
	}

	c.Start()
	time.Sleep(300 * time.Millisecond)
	logrus.Infof("将在 %s 发送Rank", c.Entry(id).Next)
}
