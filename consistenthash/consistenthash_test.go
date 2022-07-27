package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	testMap := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	testMap.Add("6", "4", "2")

	testCases := map[string]string {
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		res := testMap.Get(k)
		if res != v {
			t.Errorf("Asking for %s, should have yielded %s but %s", k, v, res)
		}
	}

	testMap.Add("8")
	testCases["27"] = "8"

	for k, v := range testCases {
		res := testMap.Get(k)
		if res != v {
			t.Errorf("Asking for %s, should have yielded %s but %s", k, v, res)
		}
	}
}