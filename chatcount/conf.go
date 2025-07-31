package chatcount

type Config struct {
	// 字体文件路径
	Font string `yaml:"font"`
	// 定时发送排行榜的时间(cron表达式)
	SendRankCron string `yaml:"send_rank_cron"`
	// 排行榜标题(定时触发)
	RankTitleTicker string `yaml:"rank_title_ticker"`
	// 热词排行榜标题
	WordRankTitleTicker string `yaml:"word_rank_title_ticker"`
	// 排行榜标题(主动触发)
	RankTitleTrigger string `yaml:"rank_title_trigger"`
	// 每日定时发送排行榜可附带的消息
	MsgWithTicker string `yaml:"msg_with_ticker"`
	// 获取用户头像的质量(1,2,3)三档
	AvatarSize int64 `yaml:"avatar_size"`
	// 停用词文件路径
	StopWordFile string `yaml:"stop_word_file"`
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
