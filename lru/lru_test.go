package lru

import (
	"reflect"
	"testing"
)

// lru_test 中的 String 实现了 lru 中的 Value 接口
type String string

func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key1", String("value1"))

	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "value1" {
		t.Fatalf("cache hit key1=value1 failed")
	}

	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveLRU(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"
	cap := len(k1 + k2 + v1 + v2)

	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("RemoveLRU key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	k1, k2, k3, k4 := "key1", "key2", "key3", "key4"
	v1, v2, v3, v4 := "value1", "value2", "value3", "value4"
	cap := len(k1 + k2 + v1 + v2)

	keys := make([]string, 0)
	callback := func(key string, val Value) {
		keys = append(keys, key)
	}

	lru := New(int64(cap), callback)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))
	lru.Add(k4, String(v4))

	expect := []string{"key1", "key2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s, but exactly %s", expect, keys)
	}
}