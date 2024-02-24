package nodemanager

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestGetBalance tests the GetBalance function of the ClientManager
func TestGetBalance(t *testing.T) {
	// Start a mock Ethereum node server
	mockServer := mockEthereumNode(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`, http.StatusOK)
	defer mockServer.Close()

	// Set the environment variable to the mock server's URL
	setEnv(t, "ETH_NODE_URL", mockServer.URL)
	defer unsetEnv(t, "ETH_NODE_URL")

	// Initialize ClientManager here, ensuring it uses ETH_NODE_URL
	httpClient := &http.Client{}
	manager := NewClientManager([]NodeConfig{{Name: "MockNode", URL: os.Getenv("ETH_NODE_URL")}}, httpClient)

	balance, err := manager.GetBalance("0xSomeEthereumAddress")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assuming the balance is returned as a hex string, convert it
	expectedBalance := "0x1" // Since the mock server responds with "0x1"
	if balance != expectedBalance {
		t.Fatalf("Expected balance %s, got %s", expectedBalance, balance)
	}
}

// Mock Ethereum node response
func mockEthereumNode(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(response))
		if err != nil {
			return
		}
	}))
}

// Helper function to set environment variables for testing
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set environment variable: %s", err)
	}
}

// Helper function to unset environment variables after tests
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset environment variable: %s", err)
	}
}
