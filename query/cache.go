package query

import (
    "sync"
)

var filterCache sync.Map // map[string]CompiledExpression

func getCachedFilter(query string) (CompiledExpression, bool) {
    if cachedFilter, ok := filterCache.Load(query); ok {
        return cachedFilter.(CompiledExpression), true
    }
    return nil, false
}

func cacheFilter(query string, compiledExpr CompiledExpression) {
    filterCache.Store(query, compiledExpr)
}
