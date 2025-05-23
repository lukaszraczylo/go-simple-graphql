package gql

import (
	"net/http/httptest"
	"testing"

	assertions "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Tests struct {
	suite.Suite
}

var (
	assert     *assertions.Assertions
	mockServer *httptest.Server
)

func (suite *Tests) SetupSuite() {
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
}

func (suite *Tests) SetupTest() {
	assert = assertions.New(suite.T())
}

func (suite *Tests) BeforeTest(suiteName, testName string) {
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Tests))
}
