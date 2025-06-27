package ctsdk

import (
	"errors"
	"github.com/kohmebot/plugin"
	"reflect"
)

type CtInvoker struct {
	v reflect.Value
}

func NewCtInvoker(env plugin.Env) (*CtInvoker, error) {
	ct, ok := env.GetPlugin("chatcount")
	if !ok {
		return nil, errors.New("chatcount not exists")
	}

	return &CtInvoker{
		v: reflect.ValueOf(ct),
	}, nil
}

func (c *CtInvoker) GetGroupChatInfo(gid int64, isOnlyToday bool) map[int64][2]int64 {
	args := []reflect.Value{
		reflect.ValueOf(gid),
		reflect.ValueOf(isOnlyToday),
	}
	return c.v.MethodByName("GetGroupChatInfo").Call(args)[0].Interface().(map[int64][2]int64)
}
