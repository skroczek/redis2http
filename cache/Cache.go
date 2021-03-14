package cache

type Cache struct {
	entries map[string]Entry
}

func NewCache() *Cache {
	cache := new(Cache)
	cache.entries = make(map[string]Entry)

	return cache
}

func (cache Cache) Has(key string) bool {
	_, ok := cache.entries[key]
	return ok
}

func (cache Cache) Key(key string) Entry {
	cacheEntry, _ := cache.entries[key]

	return cacheEntry
}

func (cache Cache) Set(key string, entry Entry) Cache {
	cache.entries[key] = entry

	return cache
}
