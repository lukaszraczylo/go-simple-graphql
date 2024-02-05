package gql

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/gookit/goutil/strutil"
)

func searchForKeysInMapStringInterface(msi map[string]interface{}, key string) (value any) {
	if msi == nil {
		return nil
	}
	return msi[key]
}

func calculateHash(query *Query) string {
	return strutil.Md5(query.JsonQuery)
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
			b.Logger.Error("Can't decode response into mapstring", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
		return result, nil
	case "string":
		return string(response), nil
	case "byte":
		return response, nil
	default:
		b.Logger.Error("Can't decode response;", map[string]interface{}{"error": "unknown response type"})
		return nil, fmt.Errorf("Can't decode response - unknown response type specified")
	}
}
