package tslist

import (
	"sync"
	"container/list"
)

type TSList struct {
	lock sync.RWMutex
	list *list.List
}
func New() *TSList {
	return &TSList{
		list : list.New(),
	}
}

func (l *TSList) Back() *list.Element {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.list.Back()
}
func (l *TSList) Front() *list.Element {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.list.Front()
}
func (l *TSList) Init() *TSList {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.Init()
	return l
}
func (l *TSList) InsertAfter(v interface{}, mark *list.Element) *list.Element {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.list.InsertAfter(v, mark)
}
func (l *TSList) InsertBefore(v interface{}, mark *list.Element) *list.Element {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.list.InsertBefore(v, mark)
}
func (l *TSList) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.list.Len()
}

func (l *TSList) MoveAfter(e, mark *list.Element) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.MoveAfter(e, mark)
}
func (l *TSList) MoveBefore(e, mark *list.Element) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.MoveAfter(e, mark)
}
func (l *TSList) MoveToBack(e *list.Element) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.MoveToBack(e)
}
func (l *TSList) MoveToFront(e *list.Element) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.MoveToFront(e)
}

func (l *TSList) PushBack(v interface{}) *list.Element {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.list.PushBack(v)
}

func (l *TSList) PushBackList(other *list.List) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.PushBackList(other)
}

func (l *TSList) PushFront(v interface{}) *list.Element {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.list.PushFront(v)
}

func (l *TSList) PushFrontList(other *list.List) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list.PushFrontList(other)
}

func (l *TSList) Remove(e *list.Element) interface{} {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.list.Remove(e)
}
