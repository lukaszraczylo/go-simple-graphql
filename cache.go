package gql

func (c *BaseClient) cacheLookup(hash string) []byte {
	if c.cache.client != nil {
		obj, found := c.cache.client.Get(hash)
		if found {
			c.Logger.Debug(c, "Cache hit;", "hash", hash)
			return obj.([]byte)
		}
		// if error is not nil it means that hash is not present in cache
		c.Logger.Debug(c, "Cache miss;", "hash", hash)
	}
	return nil
}
