package handler

import (
	"encoding/json"
	"fmt"
	"github.com/luishsr/eth-proxy/internal/nodemanager"
	"github.com/luishsr/eth-proxy/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// MockClientManager is a mock implementation of the ClientManager
type MockClientManager struct {
	Balance    string
	Err        error
	Cache      map[string]nodemanager.CacheItem
	httpClient *http.Client
	Nodes      []nodemanager.EthereumNode
}

// Provide dummy implementations for GetNodeName and IsReady to prevent panics
func (m *MockClientManager) GetNodeName() string {
	return "MockNode"
}

func (m *MockClientManager) IsReady() bool {
	return true
}

func (m *MockClientManager) GetBalance(address string) (string, error) {
	if !utils.IsValidEthereumAddress(address) {
		return "", utils.ErrInvalidAddress
	}
	return m.Balance, m.Err
}

// setEnv is a helper function for setting an environment variable for the duration of a test.
func setEnv(t *testing.T, key, value string) {
	t.Helper() // Marks this function as a test helper function.

	// Set the environment variable using os.Setenv.
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set environment variable: %s", err)
	}
}

// unsetEnv is a helper function for unsetting an environment variable for the duration of a test.
func unsetEnv(t *testing.T, key string) {
	t.Helper() // Marks this function as a test helper function.

	// Remove the environment variable using os.Unsetenv.
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset environment variable: %s", err)
	}
}

// mockEthereumNode creates a mock Ethereum node server that responds with the given response and status code.
func mockEthereumNode(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(response))
		if err != nil {
			return
		}
	}))
}

func NewClientManager(nodes []nodemanager.NodeConfig, _ *http.Client) *MockClientManager {
	// Initialize a new ClientManager instance with an empty cache and the provided HTTP client.
	manager := &MockClientManager{
		Balance: "16",
		Cache:   make(map[string]nodemanager.CacheItem),
	}

	// Iterate over the provided node configurations, creating EthereumNode instances
	// for each and adding them to the ClientManager's list of nodes.
	for _, n := range nodes {
		// Use NodeConfig (n) directly or create EthereumNode instances based on NodeConfig
		newNode := nodemanager.EthereumNode{Name: n.Name, URL: n.URL, Healthy: true}
		manager.Nodes = append(manager.Nodes, newNode)
	}

	// Return the initialized ClientManager instance.
	return manager
}

// TestGetBalanceRetry tests the retry mechanism in GetBalance method
func TestGetBalanceRetry(t *testing.T) {
	// Mock server that simulates a timeout
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay to trigger a timeout
	}))
	defer timeoutServer.Close()

	// Mock server that simulates a successful response
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x10"}`))
		if err != nil {
			return
		} // Sample response
	}))
	defer successServer.Close()

	// Initialize ClientManager with the mock servers, placing the timeout server first
	httpClient := &http.Client{Timeout: 1 * time.Second} // Set timeout less than the delay in timeoutServer
	manager := NewClientManager([]nodemanager.NodeConfig{
		{Name: "TimeoutNode", URL: timeoutServer.URL},
		{Name: "SuccessNode", URL: successServer.URL},
	}, httpClient)

	// Attempt to get balance, expecting a retry to occur and eventually succeed with the successServer
	balance, err := manager.GetBalance("0x00a3Ac5E156B4B291ceB59D019121beB6508d93D")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	// Assuming the balance is returned as a hex string, convert it and assert the value
	expectedBalance := "16" // Hex 0x10 is 16 in decimal
	if balance != expectedBalance {
		t.Fatalf("Expected balance %s, got %s", expectedBalance, balance)
	}
}

func TestProxyHandler(t *testing.T) {
	tests := []struct {
		name           string
		address        string
		mockBalance    string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid address with balance",
			address:        "0x00a3Ac5E156B4B291ceB59D019121beB6508d93D",
			mockBalance:    "100",
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"balance":"100"}`,
		},
		{
			name:           "Invalid address",
			address:        "0xInvalid",
			mockBalance:    "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid or missing Ethereum address"}`,
		},
		{
			name:           "Error fetching balance",
			address:        "0x00a3Ac5E156B4B291ceB59D019121beB6508d93D",
			mockBalance:    "",
			mockError:      fmt.Errorf("internal error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal error"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockManager := MockClientManager{
				Balance: tc.mockBalance,
				Err:     tc.mockError,
			}

			handler := NewAPIHandler(&mockManager)

			req := httptest.NewRequest("GET", fmt.Sprintf("/eth/balance/%s", tc.address), nil)
			rr := httptest.NewRecorder()

			handlerFunc := handler.ProxyHandler()
			handlerFunc.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			// Ensure the response body matches the expected JSON format
			expectedJSON := make(map[string]interface{})
			actualJSON := make(map[string]interface{})

			err := json.Unmarshal([]byte(tc.expectedBody), &expectedJSON)

			if err != nil {
				return
			}

			err = json.Unmarshal(rr.Body.Bytes(), &actualJSON)
			if err != nil {
				return
			}

			if !jsonEqual(expectedJSON, actualJSON) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tc.expectedBody)
			}
		})
	}
}

// TestGetBalance tests the GetBalance function of the ClientManager
func TestGetBalance(t *testing.T) {
	// Start a mock Ethereum node server
	mockServer := mockEthereumNode(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`, http.StatusOK)
	defer mockServer.Close()

	// Set the environment variable to the mock server's URL
	setEnv(t, "ETH_NODE_URL", mockServer.URL)
	defer unsetEnv(t, "ETH_NODE_URL")

	// Initialize ClientManager
	httpClient := &http.Client{}
	manager := NewClientManager([]nodemanager.NodeConfig{{Name: "MockNode", URL: os.Getenv("ETH_NODE_URL")}}, httpClient)

	balance, err := manager.GetBalance("0x00a3Ac5E156B4B291ceB59D019121beB6508d93D")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assuming the balance is returned as a hex string, convert it and assert the value
	expectedBalance := "16" // Since the mock server responds with "0x1"
	if balance != expectedBalance {
		t.Fatalf("Expected balance %s, got %s", expectedBalance, balance)
	}
}

// jsonEqual compares two JSON documents for semantic equality
func jsonEqual(a, b interface{}) bool {
	ajson, _ := json.Marshal(a)
	bjson, _ := json.Marshal(b)
	return string(ajson) == string(bjson)
}
