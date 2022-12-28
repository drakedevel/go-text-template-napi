package main

import (
	"container/list"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

type envStack struct {
	list *list.List
}

func newEnvStack() envStack {
	return envStack{list.New()}
}

func (es *envStack) Enter(env napi.Env) {
	es.list.PushBack(env)
}

func (es *envStack) Current() napi.Env {
	back := es.list.Back()
	if back == nil {
		panic("uh-oh")
	}
	return back.Value.(napi.Env)
}

func (es *envStack) Exit(env napi.Env) {
	back := es.list.Back()
	if back == nil || back.Value != env {
		panic("uh-oh") // XXX
	}
	es.list.Remove(back)
}
