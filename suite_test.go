package gql

import (
	"net/http/httptest"
	"os"
	"testing"

	assertions "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Tests struct {
	suite.Suite
}

var (
	assert                    *assertions.Assertions
	mockServer                *httptest.Server
	originalInsecureSkipValue string
)

func (suite *Tests) SetupSuite() {
	// Save original environment variable value
	originalInsecureSkipValue = os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY")

	// Set environment variable to skip TLS verification for test mock server
	// The mock server uses self-signed certificates
	os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", "true")

	// Start mock GraphQL server
	mockServer = StartMockServer()
}

func (suite *Tests) TearDownSuite() {
	// Close the mock server
	if mockServer != nil {
		mockServer.Close()
	}

	// Cleanup shared test cache
	CleanupTestCache()

	// Restore original environment variable value
	if originalInsecureSkipValue != "" {
		os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", originalInsecureSkipValue)
	} else {
		os.Unsetenv("GRAPHQL_INSECURE_SKIP_VERIFY")
	}
}

func (suite *Tests) SetupTest() {
	assert = assertions.New(suite.T())
}

func (suite *Tests) BeforeTest(suiteName, testName string) {
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Tests))
}
