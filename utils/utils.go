package utils

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

var (
	ErrInvalidAddress = errors.New("invalid Ethereum address")
	Logger            = logrus.New()
)

func init() {
	// Initialize log configuration
	Logger.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
}

// RespondJSON sends a JSON response with the given status code and payload
func RespondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(response)
	if err != nil {
		return
	}
}

// RespondError sends an error response in a consistent format
func RespondError(w http.ResponseWriter, statusCode int, message string) {
	RespondJSON(w, statusCode, map[string]string{"error": message})
}

// IsValidEthereumAddress checks if the provided string is a valid Ethereum address.
func IsValidEthereumAddress(address string) bool {
	// Ethereum addresses are 42 characters long and start with '0x'.
	return len(address) == 42 && strings.HasPrefix(address, "0x")
}
