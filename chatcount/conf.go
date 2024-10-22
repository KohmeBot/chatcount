package chatcount

type Config struct {
	// 字体文件路径
	Font string `mapsturcture:"font"`
	// 每日定时发送排行榜的时间 (hour)
	SendRankTime int64 `mapsturcture:"send_rank_time"`
	// 排行榜标题(定时触发)
	RankTitleTicker string `mapsturcture:"rank_title_ticker"`
	// 排行榜标题(主动触发)
	RankTitleTrigger string `mapsturcture:"rank_title_trigger"`
	// 每日定时发送排行榜可附带的消息
	MsgWithTicker string `mapsturcture:"msg_with_ticker"`
}
