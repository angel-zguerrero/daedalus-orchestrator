package rest_api_admin

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type loginRequest struct {
	UsernameOrEmail string `json:"UsernameOrEmail" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

// loginHandler handles the /admin-api/login endpoint.
func (api *RestAdminAPI) loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Warn().Err(err).Msg("Login attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	loginCmd := &commands.LoginCommand{
		UsernameOrEmail: req.UsernameOrEmail,
		Password:        req.Password,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: loginCmd,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := api.node.Read(ctx, *queryCommand)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login command execution failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error"})
		return
	}

	if parsedResult.Error != "" {
		api.logger.Error().Str("username", req.UsernameOrEmail).Str("error", parsedResult.Error).Msg("Login command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error (result type)"})
		return
	}
	loggedIn := parsedResult.Result.(bool)

	if !loggedIn {
		api.logger.Warn().Str("username", req.UsernameOrEmail).Msg("Login attempt failed: invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	tokenString, err := api.generateJWT(req.UsernameOrEmail)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to generate JWT token during login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login successful, but failed to generate token: " + err.Error()})
		return
	}

	registerSessionCmd := &commands.RegisterSessionCommand{
		JWTToken: tokenString,
		JWTKey:   api.jwtKey,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  registerSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()

	_, err = api.node.Write(writeCtx, fsmCmd)
	if err != nil {

		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to register session after login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register session after login: " + err.Error()})
		return
	}

	api.logger.Info().Str("username", req.UsernameOrEmail).Msg("User logged in successfully and session registered")
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
	})
}

// generateJWT generates a new JWT token.
// This is a placeholder and will be properly implemented in the login handler.
func (api *RestAdminAPI) generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(api.jwtDuration)
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(api.jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
