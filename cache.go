package gql

func (c *BaseClient) cacheLookup(hash string) []byte {
	if c.cache.client != nil {
		g, err := c.cache.client.Get(hash)
		if err == nil {
			c.Logger.Debug(c, "Cache hit;", "hash", hash)
			return g
		}
		// if error is not nil it means that hash is not present in cache
		c.Logger.Debug(c, "Cache miss;", "hash", hash)
	}
	return nil
}
