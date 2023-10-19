package gql

func (c *BaseClient) cacheLookup(hash string) []byte {
	if c.cache.client != nil {
		obj, found := c.cache.client.Get(hash)
		if found {
			c.Logger.Debug("Cache hit", map[string]interface{}{"hash": hash})
			return obj
		}
		// if error is not nil it means that hash is not present in cache
		c.Logger.Debug("Cache miss", map[string]interface{}{"hash": hash})
	}
	return nil
}
