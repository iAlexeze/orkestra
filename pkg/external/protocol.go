package external

import (
	"context"
	"sync"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ProtocolClient executes one external call and returns the result map to
// inject under .external.<name>. The map is used directly by the runner.
//
// HTTP clients return: status, body, error, called + auto-parsed JSON keys.
// Non-HTTP clients return: result, raw, error, called + protocol-specific keys.
type ProtocolClient interface {
	Fetch(ctx context.Context, spec orktypes.ExternalCallSpec, resolvedURL, resolvedQuery, resolvedBody, credential string) (map[string]interface{}, error)
}

func newProtocolClient(protocol orktypes.ExternalProtocol) ProtocolClient {
	switch protocol {
	case "", orktypes.ProtocolHTTP:
		return &httpProtocolClient{}
	case orktypes.ProtocolPrometheus:
		return &prometheusClient{}
	case orktypes.ProtocolRedis:
		return &redisClient{}
	case orktypes.ProtocolPostgres:
		return &postgresClient{}
	case orktypes.ProtocolMongo:
		return &mongoClient{}
	case orktypes.ProtocolKafka:
		return &kafkaClient{}
	default:
		return &httpProtocolClient{} // unknown protocol falls through to HTTP; validate catches it
	}
}

// ── Cache ─────────────────────────────────────────────────────────────────────

type cachedResult struct {
	data      map[string]interface{}
	expiresAt time.Time
}

// resultCache holds per-(gvk, call-name, url, query) cached results.
// Key format: "<gvk>/<name>/<url>/<query>".
var resultCache sync.Map

func cacheKey(gvk, name, url, query string) string {
	return gvk + "\x00" + name + "\x00" + url + "\x00" + query
}

func cacheGet(key string) (map[string]interface{}, bool) {
	v, ok := resultCache.Load(key)
	if !ok {
		return nil, false
	}
	entry := v.(cachedResult)
	if time.Now().After(entry.expiresAt) {
		resultCache.Delete(key)
		return nil, false
	}
	return entry.data, true
}

func cacheSet(key string, data map[string]interface{}, ttl time.Duration) {
	resultCache.Store(key, cachedResult{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	})
}
