package handler

import (
	"errors"
	"github.com/luishsr/eth-proxy/internal/nodemanager" // Import for accessing the ClientManagerInterface
	"github.com/luishsr/eth-proxy/utils"                // Import for utility functions like logging and responding with JSON
	"net/http"
	"strings"
)

// APIHandler holds a reference to the ClientManagerInterface to interact with Ethereum nodes.
type APIHandler struct {
	manager nodemanager.ClientManagerInterface
}

// NewAPIHandler creates a new instance of APIHandler with the provided manager.
func NewAPIHandler(manager nodemanager.ClientManagerInterface) *APIHandler {
	return &APIHandler{manager: manager}
}

// ProxyHandler returns an http.HandlerFunc that handles Ethereum balance requests.
func (api *APIHandler) ProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Extract the Ethereum address from the URL path, removing the prefix.
		address := strings.TrimPrefix(req.URL.Path, "/eth/balance/")

		// Validate the Ethereum address format.
		if address == "" || !utils.IsValidEthereumAddress(address) {
			// Respond with an error if the address is invalid or missing.
			utils.RespondError(w, http.StatusBadRequest, "Invalid or missing Ethereum address")
			return
		}

		// Attempt to retrieve the balance for the given Ethereum address.
		balance, err := api.manager.GetBalance(address)
		if err != nil {
			// Check if the error is due to an invalid address and respond accordingly.
			if errors.Is(err, utils.ErrInvalidAddress) {
				utils.RespondError(w, http.StatusBadRequest, err.Error())
			} else {
				// Log the error and respond with a generic internal server error message for any other errors.
				utils.Logger.Errorf("Error fetching balance: %v", err)
				utils.RespondError(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Respond with the retrieved balance in JSON format.
		utils.RespondJSON(w, http.StatusOK, map[string]string{"balance": balance})
	}
}
