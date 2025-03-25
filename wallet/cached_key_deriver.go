package wallet

import (
	"container/list"
	"sync"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
)

// CachedKeyDeriver is a wrapper around KeyDeriver that caches derived keys
// to improve performance for repeated derivations with the same parameters.
// It uses an LRU cache with configurable size.
type CachedKeyDeriver struct {
	keyDeriver  keyDeriverInterface
	cache       *lruCache
	maxCacheSize int
	mu          sync.Mutex
}

type cacheKey struct {
	method      string
	protocol    WalletProtocol
	keyID       string
	counterparty WalletCounterparty
	forSelf     bool
}

type cacheValue struct {
	value interface{}
	elem  *list.Element
}

type lruCache struct {
	items map[cacheKey]*cacheValue
	list  *list.List
	mu    sync.Mutex
}

// NewCachedKeyDeriver creates a new CachedKeyDeriver instance.
// rootKey is the root private key or 'anyone' key.
// maxCacheSize specifies the maximum number of items to cache (default 1000 if <= 0).
func NewCachedKeyDeriver(rootKey *ec.PrivateKey, maxCacheSize int) *CachedKeyDeriver {
	if maxCacheSize <= 0 {
		maxCacheSize = 1000
	}

	return &CachedKeyDeriver{
		keyDeriver:  NewKeyDeriver(rootKey),
		cache: &lruCache{
			items: make(map[cacheKey]*cacheValue),
			list:  list.New(),
		},
		maxCacheSize: maxCacheSize,
	}
}

// DerivePublicKey derives a public key with caching.
func (c *CachedKeyDeriver) DerivePublicKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty, forSelf bool) (*ec.PublicKey, error) {
	key := cacheKey{
		method:      "derivePublicKey",
		protocol:    protocol,
		keyID:       keyID,
		counterparty: counterparty,
		forSelf:     forSelf,
	}

	if val, ok := c.cacheGet(key); ok {
		if pubKey, ok := val.(*ec.PublicKey); ok {
			return pubKey, nil
		}
	}

	pubKey, err := c.keyDeriver.DerivePublicKey(protocol, keyID, counterparty, forSelf)
	if err != nil {
		return nil, err
	}

	c.cacheSet(key, pubKey)
	return pubKey, nil
}

// DerivePrivateKey derives a private key with caching.
func (c *CachedKeyDeriver) DerivePrivateKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty) (*ec.PrivateKey, error) {
	key := cacheKey{
		method:      "derivePrivateKey",
		protocol:    protocol,
		keyID:       keyID,
		counterparty: counterparty,
	}

	if val, ok := c.cacheGet(key); ok {
		if privKey, ok := val.(*ec.PrivateKey); ok {
			return privKey, nil
		}
	}

	privKey, err := c.keyDeriver.DerivePrivateKey(protocol, keyID, counterparty)
	if err != nil {
		return nil, err
	}

	c.cacheSet(key, privKey)
	return privKey, nil
}

// DeriveSymmetricKey derives a symmetric key with caching.
func (c *CachedKeyDeriver) DeriveSymmetricKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty) (*ec.SymmetricKey, error) {
	key := cacheKey{
		method:      "deriveSymmetricKey",
		protocol:    protocol,
		keyID:       keyID,
		counterparty: counterparty,
	}

	if val, ok := c.cacheGet(key); ok {
		if symKey, ok := val.(*ec.SymmetricKey); ok {
			return symKey, nil
		}
	}

	symKey, err := c.keyDeriver.DeriveSymmetricKey(protocol, keyID, counterparty)
	if err != nil {
		return nil, err
	}

	c.cacheSet(key, symKey)
	return symKey, nil
}

// cacheGet retrieves a value from cache and updates its LRU position.
func (c *CachedKeyDeriver) cacheGet(key cacheKey) (interface{}, bool) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	if val, ok := c.cache.items[key]; ok {
		c.cache.list.MoveToFront(val.elem)
		return val.value, true
	}
	return nil, false
}

// cacheSet adds a value to cache, evicting LRU items if needed.
func (c *CachedKeyDeriver) cacheSet(key cacheKey, value interface{}) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	// If key exists, update value and move to front
	if val, ok := c.cache.items[key]; ok {
		val.value = value
		c.cache.list.MoveToFront(val.elem)
		return
	}

	// Add new item
	elem := c.cache.list.PushFront(key)
	c.cache.items[key] = &cacheValue{
		value: value,
		elem:  elem,
	}

	// Evict if needed
	if len(c.cache.items) > c.maxCacheSize {
		oldest := c.cache.list.Back()
		if oldest != nil {
			delete(c.cache.items, oldest.Value.(cacheKey))
			c.cache.list.Remove(oldest)
		}
	}
}
