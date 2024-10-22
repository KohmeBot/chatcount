package chatcount

type Config struct {
	// 字体文件路径
	Font string `mapstructure:"font"`
	// 每日定时发送排行榜的时间 (hour)
	SendRankTime int64 `mapstructure:"send_rank_time"`
	// 排行榜标题(定时触发)
	RankTitleTicker string `mapstructure:"rank_title_ticker"`
	// 排行榜标题(主动触发)
	RankTitleTrigger string `mapstructure:"rank_title_trigger"`
	// 每日定时发送排行榜可附带的消息
	MsgWithTicker string `mapstructure:"msg_with_ticker"`
}
