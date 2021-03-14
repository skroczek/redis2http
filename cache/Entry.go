package cache

import "time"

type Entry struct {
	Hash     [32]byte
	DateTime time.Time
}

