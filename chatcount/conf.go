package chatcount

type Config struct {
	// 字体文件路径
	Font string `mapstructure:"font"`
	// 定时发送排行榜的时间(cron表达式)
	SendRankCron string `mapstructure:"send_rank_cron"`
	// 排行榜标题(定时触发)
	RankTitleTicker string `mapstructure:"rank_title_ticker"`
	// 热词排行榜标题
	WordRankTitleTicker string `mapstructure:"word_rank_title_ticker"`
	// 排行榜标题(主动触发)
	RankTitleTrigger string `mapstructure:"rank_title_trigger"`
	// 每日定时发送排行榜可附带的消息
	MsgWithTicker string `mapstructure:"msg_with_ticker"`
	// 获取用户头像的质量(1,2,3)三档
	AvatarSize int64 `mapstructure:"avatar_size"`
	// 停用词文件路径
	StopWordFile string `mapstructure:"stop_word_file"`
}

func (c *Config) AvatarSizeToParam() int {
	if c.AvatarSize <= 1 {
		return 100
	}
	if c.AvatarSize == 2 {
		return 140
	}
	return 640
}
