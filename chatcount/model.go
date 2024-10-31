package chatcount

import (
	"errors"
	"fmt"
	"github.com/fumiama/cron"
	"github.com/sirupsen/logrus"
	"github.com/yanyiwu/gojieba"
	"gorm.io/gorm"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RomiChan/syncx"
)

const (
	chatInterval = 300
)

type Word struct {
	Word  string
	Count int64
}

type WordCounter struct {
	wmp map[string]int64
}

func (w *WordCounter) Saves(words []string) {
	for _, v := range words {
		w.Save(v)
	}
}

func (w *WordCounter) Save(word string) {
	word = strings.TrimSpace(word)
	if len(word) <= 0 {
		return
	}
	w.wmp[word]++
}

func (w *WordCounter) Clear() {
	clear(w.wmp)
}

// GetWordRank 返回一个排序好的词频
func (w *WordCounter) GetWordRank(maxWord int) (res []Word) {
	kvs := make([]Word, 0, len(w.wmp))
	for k, v := range w.wmp {
		kvs = append(kvs, Word{k, v})
	}

	// 排序词频
	slices.SortFunc(kvs, func(a, b Word) int {
		return int(b.Count - a.Count)
	})

	// 确保 res 的长度不超过 kvs 的长度
	if maxWord > len(kvs) {
		maxWord = len(kvs)
	}

	res = make([]Word, maxWord)
	for i := 0; i < maxWord; i++ {
		res[i] = kvs[i]
	}
	return res
}

// chattimedb 聊天时长数据库结构体
type chattimedb struct {
	// ctdb.userTimestampMap 每个人发言的时间戳 key=groupID_userID
	userTimestampMap syncx.Map[string, int64]
	// ctdb.userTodayTimeMap 每个人今日水群时间 key=groupID_userID
	userTodayTimeMap syncx.Map[string, int64]
	// ctdb.userTodayMessageMap 每个人今日水群次数 key=groupID_userID
	userTodayMessageMap syncx.Map[string, int64]
	// ctdb.userTodayWordMap 每个人今日的热词 key=groupID_userID
	userTodayWordMap syncx.Map[string, WordCounter]
	// db 数据库
	db *gorm.DB
	// chatmu 读写添加锁
	chatmu sync.Mutex

	// 停用词
	stopWords map[string]struct{}

	jieba *gojieba.Jieba

	l *leveler
}

// initialize 初始化
func initialize(gdb *gorm.DB, stopWords []string) (*chattimedb, error) {
	stopWords = append(stopWords, filter...)
	err := gdb.AutoMigrate(&chatTime{})
	if err != nil {
		return nil, err
	}
	mp := make(map[string]struct{}, len(stopWords))
	for _, word := range stopWords {
		mp[word] = struct{}{}
	}
	logrus.Infof("已加载 %d 个停用词", len(mp))
	return &chattimedb{
		db:        gdb,
		stopWords: mp,
		jieba:     gojieba.NewJieba(),
		l:         newLeveler(60, 120, 180, 240, 300),
	}, nil
}

// chatTime 聊天时长，时间的单位都是秒
type chatTime struct {
	ID           uint   `gorm:"primary_key"`
	GroupID      int64  `gorm:"column:group_id"`
	UserID       int64  `gorm:"column:user_id"`
	TodayTime    int64  `gorm:"-"`
	TodayMessage int64  `gorm:"-"`
	TotalTime    int64  `gorm:"column:total_time;default:0"`
	TotalMessage int64  `gorm:"column:total_message;default:0"`
	HotWords     []Word `gorm:"-"`
}

// TableName 表名
func (chatTime) TableName() string {
	return "chat_time"
}

// updateChatTime 更新发言时间,todayTime的单位是分钟
func (ctdb *chattimedb) updateChatTime(gid, uid int64) (remindTime int64, remindFlag bool) {
	ctdb.chatmu.Lock()
	defer ctdb.chatmu.Unlock()
	db := ctdb.db
	now := time.Now()
	keyword := fmt.Sprintf("%v_%v", gid, uid)
	ts, ok := ctdb.userTimestampMap.Load(keyword)
	if !ok {
		ctdb.userTimestampMap.Store(keyword, now.Unix())
		ctdb.userTodayMessageMap.Store(keyword, 1)
		return
	}
	lastTime := time.Unix(ts, 0)
	todayTime, _ := ctdb.userTodayTimeMap.Load(keyword)
	totayMessage, _ := ctdb.userTodayMessageMap.Load(keyword)
	// 这个消息数是必须统计的
	ctdb.userTodayMessageMap.Store(keyword, totayMessage+1)
	st := chatTime{
		GroupID:      gid,
		UserID:       uid,
		TotalTime:    todayTime,
		TotalMessage: totayMessage,
	}

	// 如果不是同一天，把TotalTime,TotalMessage重置
	if lastTime.YearDay() != now.YearDay() {
		if err := db.Model(&st).Where("group_id = ? and user_id = ?", gid, uid).First(&st).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				db.Model(&st).Create(&st)
			}
		} else {
			db.Model(&st).Where("group_id = ? and user_id = ?", gid, uid).Updates(
				map[string]any{
					"total_time":    st.TotalTime + todayTime,
					"total_message": st.TotalMessage + totayMessage,
				})
		}
		ctdb.userTimestampMap.Store(keyword, now.Unix())
		ctdb.userTodayTimeMap.Delete(keyword)
		ctdb.userTodayMessageMap.Delete(keyword)
		return
	}

	userChatTime := int64(now.Sub(lastTime).Seconds())
	// 当聊天时间在一定范围内的话，则计入时长
	if userChatTime < chatInterval {
		ctdb.userTodayTimeMap.Store(keyword, todayTime+userChatTime)
		remindTime = (todayTime + userChatTime) / 60
		remindFlag = ctdb.l.level(int((todayTime+userChatTime)/60)) > ctdb.l.level(int(todayTime/60))
	}
	ctdb.userTimestampMap.Store(keyword, now.Unix())
	return
}

// updateChatWord 更新发言词
func (ctdb *chattimedb) updateChatWord(gid, uid int64, msgs []string) {
	ctdb.chatmu.Lock()
	defer ctdb.chatmu.Unlock()
	x := ctdb.jieba
	var words []string
	for _, msg := range msgs {
		words = append(words, x.Cut(msg, true)...)
	}
	words = slices.DeleteFunc(words, func(s string) bool {
		_, ok := ctdb.stopWords[strings.TrimSpace(s)]
		return ok
	})

	key := fmt.Sprintf("%d_%d", gid, uid)
	c, ok := ctdb.userTodayWordMap.Load(key)
	if !ok {
		c = WordCounter{map[string]int64{}}
	}
	c.Saves(words)
	ctdb.userTodayWordMap.Store(key, c)
}

// getChatTime 获得用户聊天时长和消息次数,todayTime,totalTime的单位是秒,todayMessage,totalMessage单位是条数
func (ctdb *chattimedb) getChatTime(gid, uid int64) (todayTime, todayMessage, totalTime, totalMessage int64) {
	ctdb.chatmu.Lock()
	defer ctdb.chatmu.Unlock()
	db := ctdb.db
	st := chatTime{}
	db.Model(&st).Where("group_id = ? and user_id = ?", gid, uid).First(&st)
	keyword := fmt.Sprintf("%v_%v", gid, uid)
	todayTime, _ = ctdb.userTodayTimeMap.Load(keyword)
	todayMessage, _ = ctdb.userTodayMessageMap.Load(keyword)
	totalTime = st.TotalTime
	totalMessage = st.TotalMessage
	return
}

// getChatRank 获得水群排名，时间单位为秒
func (ctdb *chattimedb) getChatRank(gid int64) (chatTimeList []chatTime) {
	ctdb.chatmu.Lock()
	defer ctdb.chatmu.Unlock()
	chatTimeList = make([]chatTime, 0, 100)
	keyList := make([]string, 0, 100)
	now := time.Now()
	ctdb.userTimestampMap.Range(func(key string, value int64) bool {
		t := time.Unix(value, 0)
		if strings.Contains(key, strconv.FormatInt(gid, 10)) && t.YearDay() == now.YearDay() {
			keyList = append(keyList, key)
		}
		return true
	})
	for _, v := range keyList {
		_, a, _ := strings.Cut(v, "_")
		uid, _ := strconv.ParseInt(a, 10, 64)
		todayTime, _ := ctdb.userTodayTimeMap.Load(v)
		todayMessage, _ := ctdb.userTodayMessageMap.Load(v)
		todayWords, _ := ctdb.userTodayWordMap.Load(v)
		chatTimeList = append(chatTimeList, chatTime{
			GroupID:      gid,
			UserID:       uid,
			TodayTime:    todayTime,
			TodayMessage: todayMessage,
			HotWords:     todayWords.GetWordRank(5),
		})
	}
	sort.Sort(sortChatTime(chatTimeList))
	return
}

func (ctdb *chattimedb) autoClear() error {
	c := cron.New()
	_, err := c.AddFunc("0 0 * * *", func() {
		time.Sleep(15 * time.Second)
		ctdb.chatmu.Lock()
		defer ctdb.chatmu.Unlock()
		start := time.Now()
		defer func() {
			logrus.Infof("AutoClear cost %s", time.Since(start))
		}()
		ctdb.userTimestampMap = syncx.Map[string, int64]{}
		ctdb.userTodayTimeMap = syncx.Map[string, int64]{}
		ctdb.userTodayMessageMap = syncx.Map[string, int64]{}
		ctdb.userTodayWordMap = syncx.Map[string, WordCounter]{}
		runtime.GC()
	})
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

// leveler 结构体，包含一个 levelArray 字段
type leveler struct {
	levelArray []int
}

// newLeveler 构造函数，用于创建 Leveler 实例
func newLeveler(levels ...int) *leveler {
	return &leveler{
		levelArray: levels,
	}
}

// level 方法，封装了 getLevel 函数的逻辑
func (l *leveler) level(t int) int {
	for i := len(l.levelArray) - 1; i >= 0; i-- {
		if t >= l.levelArray[i] {
			return i + 1
		}
	}
	return 0
}

// sortChatTime chatTime排序数组
type sortChatTime []chatTime

// Len 实现 sort.Interface
func (a sortChatTime) Len() int {
	return len(a)
}

// Less 实现 sort.Interface，按 TodayTime 降序，TodayMessage 降序
func (a sortChatTime) Less(i, j int) bool {
	if a[i].TodayTime == a[j].TodayTime {
		return a[i].TodayMessage > a[j].TodayMessage
	}
	return a[i].TodayTime > a[j].TodayTime
}

// Swap 实现 sort.Interface
func (a sortChatTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
