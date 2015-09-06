package store

import "qudao.com/tech/netio/syncmap"

type MemoryStore struct {
	data *syncmap.SyncMap
}
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: syncmap.New(),
	}
}

func (m *MemoryStore) Del(key string) {
	m.data.Delete(key)
}

func (m *MemoryStore) Get(key string) string{
	val, ok := m.data.Get(key)
	if ok { return  val.(string) }
	return ""
}

func (m *MemoryStore) Set(key, val string) string{
	m.data.Set(key, val)
	return key
}

func (m *MemoryStore) Has(key string) bool{
	return m.data.Has(key)
}