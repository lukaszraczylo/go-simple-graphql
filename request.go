package gql

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/avast/retry-go/v4"
)

func (c *BaseClient) executeQuery(query []byte, headers any) (result any, err error) {
	var queryResult queryResults
	httpRequest, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(query))
	if err != nil {
		c.Logger.Error(c, "Error while creating http request;", "error", err.Error())
		return
	}

	for key, value := range headers.(map[string]interface{}) {
		httpRequest.Header.Add(key, fmt.Sprintf("%s", value))
	}

	var httpResponse *http.Response

	err = retry.Do(
		func() error {
			httpResponse, err = c.client.Do(httpRequest)
			if err != nil {
				c.Logger.Error(c, "Error while executing http request;", "error", err.Error())
				return err
			}
			defer httpResponse.Body.Close()

			if httpResponse.StatusCode <= 200 && httpResponse.StatusCode >= 204 {
				return err
			}

			body, err := ioutil.ReadAll(httpResponse.Body)
			if err != nil {
				c.Logger.Error(c, "Error while reading http response;", "error", err.Error())
				return err
			}

			err = json.Unmarshal(body, &queryResult)
			if err != nil {
				c.Logger.Error(c, "Error while unmarshalling http response;", "error", err.Error())
				return err
			}

			return nil
		},
	)

	if len(queryResult.Errors) > 0 {
		return nil, fmt.Errorf("%v", queryResult.Errors)
	}
	return queryResult.Data, nil
}
