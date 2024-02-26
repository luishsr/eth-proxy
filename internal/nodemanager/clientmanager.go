package nodemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/luishsr/eth-proxy/utils"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type NodeConfig struct {
	Name string
	URL  string
}

type EthereumNode struct {
	URL        string
	Name       string
	Healthy    bool
	LastUsed   time.Time
	ErrorCount int
}

type CacheItem struct {
	Balance   string
	Timestamp time.Time
}

type ClientManager struct {
	Nodes        []*EthereumNode
	mu           sync.Mutex
	index        int
	lastNodeName string
	Cache        map[string]CacheItem
	httpClient   *http.Client
}

type jsonRPCPayload struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCResponse struct {
	Result string `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ID int `json:"id"`
}

// NewClientManager initializes a new ClientManager with the given node configurations and HTTP client.
func NewClientManager(nodes []NodeConfig, httpClient *http.Client) *ClientManager {
	manager := &ClientManager{
		Cache:      make(map[string]CacheItem),
		httpClient: httpClient,
	}

	for _, n := range nodes {
		manager.Nodes = append(manager.Nodes, &EthereumNode{Name: n.Name, URL: n.URL, Healthy: true})
	}

	return manager
}

// NextNode selects the next healthy node using a round-robin algorithm.
func (m *ClientManager) NextNode() *EthereumNode {
	m.mu.Lock()
	defer m.mu.Unlock()

	startIdx := m.index
	for attempt := 0; attempt < len(m.Nodes); attempt++ {
		node := m.Nodes[m.index]
		m.index = (m.index + 1) % len(m.Nodes)

		if node.Healthy {
			m.lastNodeName = node.Name
			return node
		}

		if m.index == startIdx {
			utils.Logger.Warn("All Ethereum nodes have been checked and none are healthy")
			break
		}

	}

	return nil // No healthy nodes found
}

// CheckNodeHealth performs a health check on the specified node.
func (m *ClientManager) CheckNodeHealth(node *EthereumNode) {
	payload := jsonRPCPayload{
		JSONRPC: "2.0",
		Method:  "web3_clientVersion",
		Params:  []interface{}{},
		ID:      1,
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		utils.Logger.WithError(err).Error("Failed to marshal JSON RPC payload")
		return
	}

	req, err := http.NewRequest("POST", node.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		utils.Logger.WithError(err).Error("Failed to create new HTTP request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)

	utils.Logger.Info("Health-checking Node: " + node.Name)

	if err != nil || resp.StatusCode != http.StatusOK {
		node.Healthy = false
		node.ErrorCount++
		if node.ErrorCount >= 3 {
			go m.cooldownNode(node, 1*time.Minute)
		}

		utils.Logger.Info("*** Node " + node.Name + " is not running!")

		utils.Logger.WithFields(logrus.Fields{
			"node":        node.Name,
			"status_code": resp.StatusCode,
			"error":       err,
		}).Println("Ethereum Node health check failed")
	} else {
		utils.Logger.Info("Node " + node.Name + " is up and running!")
		node.Healthy = true
		node.ErrorCount = 0
	}

	err = resp.Body.Close()
	if err != nil {
		return
	}
}

// cooldownNode temporarily marks a node as unhealthy before rechecking its health.
func (m *ClientManager) cooldownNode(node *EthereumNode, duration time.Duration) {
	time.Sleep(duration) // Wait for the cooldown period
	node.Healthy = true  // Assume the node might be healthy now
	node.ErrorCount = 0  // Reset error count
	utils.Logger.WithField("node", node.Name).Warn("Ethereum Node cooldown period ended, marking as healthy")
}

// GetNodeName returns the name of the last used node.
func (m *ClientManager) GetNodeName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastNodeName
}

// StartHealthChecks begins periodic health checks for each node.
func (m *ClientManager) StartHealthChecks(interval time.Duration) {
	utils.Logger.Info("Ethereum Nodes periodic health check started")
	for _, node := range m.Nodes {
		go func(n *EthereumNode) {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				m.CheckNodeHealth(n)
			}
		}(node)
	}
}

// IsReady checks if at least one node is healthy and ready.
func (m *ClientManager) IsReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, node := range m.Nodes {
		if node.Healthy {
			return true
			// At least one node is healthy
		}
	}
	utils.Logger.Println("No Ethereum Nodes Ready!")
	return false // No healthy nodes
}

// GetBalance fetches the balance for a given Ethereum address, using cache when possible, and retries with a different node if necessary.
func (m *ClientManager) GetBalance(address string) (string, error) {
	// Read timeout value from environment variable, with a default.
	timeoutSecs, err := strconv.Atoi(os.Getenv("NODE_REQUEST_TIMEOUT_SECONDS"))
	if err != nil || timeoutSecs <= 0 {
		timeoutSecs = 5 // Default timeout of 5 seconds if not specified or invalid.
	}

	// Read the max retry count from environment, with a default.
	maxRetries, err := strconv.Atoi(os.Getenv("MAX_RETRIES"))
	if err != nil || maxRetries < 0 {
		maxRetries = 3 // Default to 3 retries if not specified or invalid.
	}

	m.mu.Lock()
	cachedItem, found := m.Cache[address]
	m.mu.Unlock()

	cacheExpirationSecs, err := strconv.Atoi(os.Getenv("CACHE_EXPIRATION_SECONDS"))
	if err != nil || cacheExpirationSecs <= 0 {
		cacheExpirationSecs = 60 // Default to 60 seconds if not specified or invalid
	}

	// Check if the address is in the cache and if the cache item is still valid
	if found {
		// Calculate the age of the cache item
		cacheAge := time.Since(cachedItem.Timestamp)

		if cacheAge.Seconds() <= float64(cacheExpirationSecs) {
			// Cache item is still valid, return the cached balance
			return cachedItem.Balance, nil
		}
	}

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		node := m.NextNode()

		// No Ethereum nodes available
		if node == nil {
			return "",
				fmt.Errorf("no healthy Ethereum Nodes available to fetch the balance")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)

		balance, err := m.fetchBalanceFromNode(ctx, node, address)
		if err == nil {
			cancel()
			return balance, nil
		} else {
			m.mu.Lock()
			m.Cache[address] = CacheItem{
				Balance:   balance,
				Timestamp: time.Now(),
			}
			m.mu.Unlock()
		}

		lastErr = err
		// Mark the node as unhealthy if there was an error fetching the balance.
		node.Healthy = false

		cancel()
	}

	// Return the last error after exhausting retries.
	return "", fmt.Errorf("failed to fetch balance after %d retries, last error: %w", maxRetries, lastErr)
}

// fetchBalanceFromNode retrieves the balance for a given Ethereum address from a specific node.
func (m *ClientManager) fetchBalanceFromNode(ctx context.Context, node *EthereumNode, address string) (string, error) {
	payload := jsonRPCPayload{
		JSONRPC: "2.0",
		Method:  "eth_getBalance",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		utils.Logger.WithError(err).Error("Failed to marshal JSON RPC payload")
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", node.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		utils.Logger.WithError(err).Error("Failed to create new HTTP request")
		return "", err
	}

	// Send the request using httpClient...
	resp, err := m.httpClient.Do(req)
	if err != nil {
		utils.Logger.WithError(err).WithFields(logrus.Fields{
			"node_url": node.URL,
		}).Error("Failed to execute HTTP request")
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	// Handle response...
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		utils.Logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"node_url":    node.URL,
		}).Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	var result jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("error response from node: %s", result.Error.Message)
	}

	return result.Result, nil
}
