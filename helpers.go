package gql

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

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
	hash := md5.Sum(query.JsonQuery)
	return hex.EncodeToString(hash[:])
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
		return nil, fmt.Errorf("Can't decode response - unknown response type specified")
	}
}
