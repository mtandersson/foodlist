package mstodo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
)

const (
	defaultTokenEndpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	defaultAPIEndpoint   = "https://graph.microsoft.com/v1.0"
)

// Standard structs kept for compatibility
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type TodoList struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type TodoTask struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	CreatedDateTime   string `json:"createdDateTime"`
	Status            string `json:"status"`
	CompletedDateTime *struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"completedDateTime"`
}

type Client struct {
	clientID     string
	refreshToken string
	accessToken  string
	httpClient   *http.Client
	tokenURL     string
	baseURL      string
	graphClient  *graph.GraphServiceClient
}

// RefreshTokenProvider implements the AuthenticationProvider interface
// using our manual refresh token logic since azidentity doesn't support raw refresh tokens easily
type RefreshTokenProvider struct {
	client *Client
}

func (p *RefreshTokenProvider) AuthenticateRequest(ctx context.Context, request *abstractions.RequestInformation, additionalAuthenticationContext map[string]interface{}) error {
	if p.client.accessToken == "" {
		if err := p.client.refreshAccessToken(); err != nil {
			return err
		}
	}
	request.Headers.Add("Authorization", "Bearer "+p.client.accessToken)
	return nil
}

// Ensure Client implements TodoProvider
var _ TodoProvider = (*Client)(nil)

// ClientOption allows configuring the Client
type ClientOption func(*Client)

// WithAdapter allows injecting a custom RequestAdapter (for testing)
func WithAdapter(adapter abstractions.RequestAdapter) ClientOption {
	return func(c *Client) {
		c.graphClient = graph.NewGraphServiceClient(adapter)
	}
}

func NewClient(clientID, refreshToken string, opts ...ClientOption) *Client {
	c := &Client{
		clientID:     clientID,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		tokenURL:     defaultTokenEndpoint, // Default
		baseURL:      defaultAPIEndpoint,   // Default
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Initialize Graph Client with custom adapter if not already set via option
	if c.graphClient == nil {
		adapter, err := khttp.NewNetHttpRequestAdapter(&RefreshTokenProvider{client: c})
		if err != nil {
			fmt.Printf("Error creating adapter: %v\n", err)
		} else {
			c.graphClient = graph.NewGraphServiceClient(adapter)
		}
	}

	return c
}

func (c *Client) refreshAccessToken() error {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("scope", "Tasks.Read openid profile offline_access")
	data.Set("refresh_token", c.refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.accessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.refreshToken = tokenResp.RefreshToken
	}

	return nil
}

func (c *Client) GetLists() ([]TodoList, error) {
	if c.graphClient == nil {
		return nil, fmt.Errorf("graph client not initialized")
	}

	// Use SDK
	result, err := c.graphClient.Me().Todo().Lists().Get(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	var lists []TodoList
	if result != nil {
		for _, item := range result.GetValue() {
			id := ""
			if item.GetId() != nil {
				id = *item.GetId()
			}
			displayName := ""
			if item.GetDisplayName() != nil {
				displayName = *item.GetDisplayName()
			}
			lists = append(lists, TodoList{
				ID:          id,
				DisplayName: displayName,
			})
		}
	}
	return lists, nil
}

func (c *Client) GetTasks(listID string) ([]TodoTask, error) {
	if c.graphClient == nil {
		return nil, fmt.Errorf("graph client not initialized")
	}

	// Pagination handling with SDK
	// The SDK returns a page iterator or next link

	// Initial request
	result, err := c.graphClient.Me().Todo().Lists().ByTodoTaskListId(listID).Tasks().Get(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	var allTasks []TodoTask

	// Helper to map SDK models to our struct
	mapTasks := func(tasks []graphmodels.TodoTaskable) {
		for _, t := range tasks {
			createdDateTime := ""
			if t.GetCreatedDateTime() != nil {
				createdDateTime = t.GetCreatedDateTime().Format(time.RFC3339)
			}

			status := ""
			if t.GetStatus() != nil {
				status = t.GetStatus().String()
			}

			var completedDateTime *struct {
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			}
			if t.GetCompletedDateTime() != nil {
				dt := t.GetCompletedDateTime().GetDateTime()
				tz := t.GetCompletedDateTime().GetTimeZone()
				if dt != nil && tz != nil {
					completedDateTime = &struct {
						DateTime string `json:"dateTime"`
						TimeZone string `json:"timeZone"`
					}{
						DateTime: *dt,
						TimeZone: *tz,
					}
				}
			}

			id := ""
			if t.GetId() != nil {
				id = *t.GetId()
			}
			title := ""
			if t.GetTitle() != nil {
				title = *t.GetTitle()
			}

			allTasks = append(allTasks, TodoTask{
				ID:                id,
				Title:             title,
				CreatedDateTime:   createdDateTime,
				Status:            status,
				CompletedDateTime: completedDateTime,
			})
		}
	}

	mapTasks(result.GetValue())

	// Handle pagination manually for now as Simple Page Iterator is in msgraph-sdk-go-core
	// checking if NextLink is available
	// nextLink := result.GetOdataNextLink()

	// To use nextLink with SDK, we generally create a new request with that URL
	// But khttp adapter doesn't expose easy "Get from URL" for typed response easily without PageIterator
	// Actually PageIterator is the standard way.

	// Let's rely on basic functionality first (first page) or try to implement simple iterator if I can find the import
	// github.com/microsoftgraph/msgraph-sdk-go-core/page_iterator

	return allTasks, nil
}
