// mock_server.go
package gql

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"golang.org/x/net/http2"
)

func StartMockServer() *httptest.Server {
	// Create your handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Your existing handler code
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var req struct {
			Variables map[string]interface{} `json:"variables"`
			Query     string                 `json:"query"`
		}
		err = json.Unmarshal(bodyBytes, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Simple validation to check for unbalanced braces as a syntax error
		if !strings.HasPrefix(strings.TrimSpace(req.Query), "query") || !strings.HasSuffix(strings.TrimSpace(req.Query), "}") {
			resp := []byte(`{"errors":[{"message":"Syntax Error: Invalid query"}]}`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(resp)
			return
		}

		var resp []byte
		if strings.Contains(req.Query, "viewer") {
			resp = []byte(`{"data":{"viewer":{"login":"mockuser"}}}`)
		} else if strings.Contains(req.Query, "dragons") {
			resp = []byte(`{"data":{"dragons":[{"name":"Mock Dragon 1"},{"name":"Mock Dragon 2"}]}}`)
		} else {
			resp = []byte(`{"errors":[{"message":"Unknown query"}]}`)
			w.WriteHeader(http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})

	// Create an unstarted server
	server := httptest.NewUnstartedServer(handler)

	// Configure the server to support HTTP/2
	http2.ConfigureServer(server.Config, &http2.Server{})
	server.TLS = server.Config.TLSConfig

	// Start the server with TLS
	server.StartTLS()
	return server
}
