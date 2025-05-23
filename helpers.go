package gql

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"

	"github.com/goccy/go-json"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

func searchForKeysInMapStringInterface(msi map[string]interface{}, key string) interface{} {
	if msi == nil {
		return nil
	}
	return msi[key]
}

func calculateHash(query *Query) string {
	hash := fnv.New64a()
	hash.Write(query.JsonQuery)
	return hex.EncodeToString(hash.Sum(nil))
}

func (b *BaseClient) cacheLookup(hash string) []byte {
	obj, _ := b.cache.Get(hash)
	return obj
}

func (b *BaseClient) decodeResponse(response []byte) (any, error) {
	switch b.responseType {
	case "mapstring":
		var result map[string]interface{}
		err := json.Unmarshal(response, &result)
		if err != nil {
			b.Logger.Error(&libpack_logger.LogMessage{
				Message: "Can't decode response into mapstring",
				Pairs:   map[string]interface{}{"error": err.Error()},
			})
			return nil, err
		}
		return result, nil
	case "string":
		return string(response), nil
	case "byte":
		return response, nil
	default:
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't decode response",
			Pairs:   map[string]interface{}{"error": "unknown response type"},
		})
		return nil, fmt.Errorf("can't decode response - unknown response type specified")
	}
}
