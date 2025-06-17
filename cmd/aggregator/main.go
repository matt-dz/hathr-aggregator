package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"play-aggregator/internal/httpclient"
	"play-aggregator/internal/logging"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
)

type ListUsersResponse struct {
	IDs  []uuid.UUID `json:"ids"`
	Next uuid.UUID   `json:"next"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateSpotifyPlays struct {
	After time.Time `json:"after"`
}

func decodeJson(dst interface{}, decoder *json.Decoder) error {
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// Ensure no extra tokens after decoding
	var dud interface{}
	if err := decoder.Decode(&dud); err != io.EOF {
		return err
	}
	return nil
}

func updatePlays(userID uuid.UUID, after time.Time, bearerToken string, httpclient *httpclient.Client, logger *slog.Logger) {
	logger = logger.With(slog.String("userID", userID.String()))
	logger.Debug("Building update plays request")
	url := fmt.Sprintf("%s/api/users/%s/plays/spotify", backendUrl, userID.String())
	requestBody := UpdateSpotifyPlays{
		After: after,
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("Failed to marshal request", slog.Any("error", err))
		return
	}
	req, err := retryablehttp.NewRequest(http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		logger.Error("Failed to create request", slog.Any("error", err))
		return
	}
	req.Header.Add("Authorization", bearerToken)

	logger.Info("Sending update plays request")
	res, err := httpclient.Do(req)
	if err != nil {
		logger.Error("Failed to send request", slog.Any("error", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		logger.Error("Failed to update plays", slog.Int("statusCode", res.StatusCode), slog.String("body", string(body)))
		return
	}

	logger.Info("Successfully updated plays for user")
}

var backendUrl = os.Getenv("HATHR_BACKEND_URL")

func main() {
	currentTime := time.Now()
	client := httpclient.New()
	logger := slog.New(&logging.ContextHandler{
		Handler: slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
	})
	client.Logger = logger

	hathrUsername := os.Getenv("HATHR_USERNAME")
	hathrPassword := os.Getenv("HATHR_PASSWORD")
	if backendUrl == "" || hathrUsername == "" || hathrPassword == "" {
		log.Fatal("Please set BACKEND_URL, HATHR_USERNAME, and HATHR_PASSWORD environment variables")
	}

	logger.Info("Logging in to Hathr backend")
	requestBody := LoginRequest{
		Username: hathrUsername,
		Password: hathrPassword,
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatalf("Failed to marshal login request: %v", err)
	}

	res, err := client.Post(fmt.Sprintf("%s/api/login/admin", backendUrl), "application/json", bytes.NewReader(rawBody))
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Failed to read response body: %v", err)
		}
		log.Fatalf("Login failed with status %d: %s", res.StatusCode, body)
	}

	var loginResponse LoginResponse
	decoder := json.NewDecoder(res.Body)
	decoder.DisallowUnknownFields()
	if err := decodeJson(&loginResponse, decoder); err != nil {
		log.Fatalf("Failed to decode login response: %v", err)
	}
	bearerToken := res.Header.Get("Authorization")
	// refreshToken := loginResponse.RefreshToken

	next := uuid.Nil
	for {
		// Fetch users
		logger.Info("Fetching users", slog.String("next", next.String()))
		req, err := retryablehttp.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/users?limit=100&after=%s", backendUrl, next.String()), nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Add("Authorization", bearerToken)
		res, err = client.Do(req)
		if err != nil {
			log.Fatalf("Failed to fetch users: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalf("Failed to read response body: %v", err)
			}
			log.Fatalf("Failed to fetch users with status %d: %s", res.StatusCode, body)
		}

		decoder = json.NewDecoder(res.Body)
		decoder.DisallowUnknownFields()
		var usersResponse ListUsersResponse
		if err := decodeJson(&usersResponse, decoder); err != nil {
			log.Fatalf("Failed to decode users response: %v", err)
		}

		if len(usersResponse.IDs) == 0 {
			logger.Info("No more users to process. Exiting.")
			os.Exit(0)
		}
		next = usersResponse.Next
		logger.Info("Fetched users", slog.Int("count", len(usersResponse.IDs)), slog.String("next", next.String()))

		var wg sync.WaitGroup
		wg.Add(len(usersResponse.IDs))
		logger.Info("Processing users", slog.Int("count", len(usersResponse.IDs)))
		for _, userID := range usersResponse.IDs {
			go func() {
				defer wg.Done()
				updatePlays(userID, currentTime, bearerToken, client, logger)
			}()
		}

		wg.Wait()
	}
}
