package main

import (
	"github.com/joho/godotenv"
	"github.com/luishsr/eth-proxy/internal/handler"
	"github.com/luishsr/eth-proxy/internal/nodemanager"
	"github.com/luishsr/eth-proxy/utils"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	// Define a Prometheus counter to track API calls per Ethereum node.
	apiCallsPerNode = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_proxy_api_calls_per_node_total",
			Help: "Total number of API calls to the proxy per Ethereum node",
		},
		[]string{"node"},
	)
)

type Server struct {
	manager nodemanager.ClientManagerInterface // Interface abstraction for Ethereum node.
}

// NewServer constructs a new Server instance with a given Ethereum node manager.
func NewServer(manager nodemanager.ClientManagerInterface) *Server {
	return &Server{manager: manager}
}

// handleEthBalance processes Ethereum balance requests via the /eth/balance/ endpoint.
func (s *Server) handleEthBalance(w http.ResponseWriter, r *http.Request) {
	// Extract Ethereum address from the request path.
	address := strings.TrimPrefix(r.URL.Path, "/eth/balance/")

	// Validate Ethereum address format.
	if !utils.IsValidEthereumAddress(address) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid Ethereum address")
		return
	}

	// Delegate the request to the handler's ProxyHandler function.
	handlerFunc := handler.NewAPIHandler(s.manager).ProxyHandler()
	handlerFunc.ServeHTTP(w, r)
}

// handleHealthz provides a simple health check endpoint.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleReady checks if the service is ready to handle requests.
func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	if s.manager.IsReady() {
		w.WriteHeader(http.StatusOK)
	} else {
		utils.Logger.Println("Service Not ready")
		http.Error(w, "Service Not ready", http.StatusServiceUnavailable)
	}
}

func main() {
	// Register the API calls counter with Prometheus.
	customRegistry := prometheus.NewRegistry()
	customRegistry.MustRegister(apiCallsPerNode)

	// Load environment variables from a .env file in non-production environments.
	if _, err := os.Stat(".env"); err == nil && os.Getenv("GO_ENV") != "production" {
		if err := godotenv.Load(".env"); err != nil {
			utils.Logger.Fatal("Error loading .env file")
		}
	}

	// Initialize the ClientManager with appropriate configuration.
	httpClient := &http.Client{Timeout: 10 * time.Second}
	manager := nodemanager.NewClientManager(nil, httpClient) // Update to pass actual node configurations.

	// Start periodic health checks for Ethereum nodes.
	//manager.StartHealthChecks(30 * time.Second)

	// Start the HTTP server with JWT Middleware
	server := NewServer(manager)
	http.Handle("/eth/balance/", http.HandlerFunc(server.handleEthBalance))
	http.HandleFunc("/healthz", server.handleHealthz)
	http.HandleFunc("/ready", server.handleReady)

	// Start the HTTP server.
	utils.Logger.Println("Starting Ethereum proxy server on :8088...")
	if err := http.ListenAndServe(":8088", nil); err != nil {
		utils.Logger.Fatal(err)
	}
}
