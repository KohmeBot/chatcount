package chatcount

type Config struct {
	// 字体文件路径
	Font string `mapsturcture:"font"`
	// 每日定时发送排行榜的时间 (hour)
	SendRankTime int64 `mapsturcture:"send_rank_time"`
	// 排行榜标题
	RankTitle string `mapsturcture:"rank_title"`
}
