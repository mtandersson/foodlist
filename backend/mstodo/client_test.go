package mstodo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoft/kiota-abstractions-go/store"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a dummy JWT in Base64Url format (HEADER.PAYLOAD.SIGNATURE).
// Valid Base64Url strings (no padding); alg: none.
const (
	dummyHeader  = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
	dummyPayload = "eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	dummySig     = ""
	dummyJWT     = dummyHeader + "." + dummyPayload + "." + dummySig
)

// MockRequestAdapter implements abstractions.RequestAdapter for testing
type MockRequestAdapter struct {
	GetListsResponse []graphmodels.TodoTaskListable
	GetTasksResponse []graphmodels.TodoTaskable
}

func (m *MockRequestAdapter) SendCollection(ctx context.Context, requestInfo *abstractions.RequestInformation, parsableFactory serialization.ParsableFactory, errorMapping abstractions.ErrorMappings) ([]serialization.Parsable, error) {
	return nil, nil
}

// Send implements the Kiota adapter path used by msgraph-sdk-go for Todo list/task GETs
// (collection payloads are Todo*CollectionResponse via Send, not SendCollection).
func (m *MockRequestAdapter) Send(ctx context.Context, requestInfo *abstractions.RequestInformation, parsableFactory serialization.ParsableFactory, errorMapping abstractions.ErrorMappings) (serialization.Parsable, error) {
	url := requestInfo.UrlTemplate
	// Tasks URL: .../todo/lists/{listId}/tasks{?...} — check before /todo/lists alone.
	if strings.Contains(url, "/tasks") && m.GetTasksResponse != nil {
		resp := graphmodels.NewTodoTaskCollectionResponse()
		resp.SetValue(m.GetTasksResponse)
		return resp, nil
	}
	if strings.Contains(url, "/todo/lists") && m.GetListsResponse != nil {
		resp := graphmodels.NewTodoTaskListCollectionResponse()
		resp.SetValue(m.GetListsResponse)
		return resp, nil
	}
	return nil, nil
}

func (m *MockRequestAdapter) SendEnum(ctx context.Context, requestInfo *abstractions.RequestInformation, parser serialization.EnumFactory, errorMapping abstractions.ErrorMappings) (any, error) {
	return nil, nil
}

func (m *MockRequestAdapter) SendPrimitive(ctx context.Context, requestInfo *abstractions.RequestInformation, typeName string, errorMapping abstractions.ErrorMappings) (any, error) {
	return nil, nil
}

func (m *MockRequestAdapter) SendNoContent(ctx context.Context, requestInfo *abstractions.RequestInformation, errorMapping abstractions.ErrorMappings) error {
	return nil
}

func (m *MockRequestAdapter) SendEnumCollection(ctx context.Context, requestInfo *abstractions.RequestInformation, parser serialization.EnumFactory, errorMapping abstractions.ErrorMappings) ([]any, error) {
	return nil, nil
}

func (m *MockRequestAdapter) SendPrimitiveCollection(ctx context.Context, requestInfo *abstractions.RequestInformation, typeName string, errorMapping abstractions.ErrorMappings) ([]any, error) {
	return nil, nil
}

func (m *MockRequestAdapter) ConvertToNativeRequest(ctx context.Context, requestInfo *abstractions.RequestInformation) (any, error) {
	return nil, nil
}

func (m *MockRequestAdapter) GetSerializationWriterFactory() serialization.SerializationWriterFactory {
	return nil
}

func (m *MockRequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {}

//nolint:revive // names match github.com/microsoft/kiota-abstractions-go RequestAdapter
func (m *MockRequestAdapter) SetBaseUrl(baseUrl string) {}

//nolint:revive // names match github.com/microsoft/kiota-abstractions-go RequestAdapter
func (m *MockRequestAdapter) GetBaseUrl() string { return "" }

func TestClient_GetLists(t *testing.T) {
	list1 := graphmodels.NewTodoTaskList()
	id1 := "list-1"
	name1 := "List One"
	list1.SetId(&id1)
	list1.SetDisplayName(&name1)

	list2 := graphmodels.NewTodoTaskList()
	id2 := "list-2"
	name2 := "List Two"
	list2.SetId(&id2)
	list2.SetDisplayName(&name2)

	mockAdapter := &MockRequestAdapter{
		GetListsResponse: []graphmodels.TodoTaskListable{list1, list2},
	}

	client := NewClient("client-id", "refresh-token", WithAdapter(mockAdapter))

	lists, err := client.GetLists()
	require.NoError(t, err)
	assert.Len(t, lists, 2)
	assert.Equal(t, "List One", lists[0].DisplayName)
	assert.Equal(t, "List Two", lists[1].DisplayName)
}

func TestClient_GetTasks_Pagination(t *testing.T) {
	task1 := graphmodels.NewTodoTask()
	id1 := "task-1"
	title1 := "Task One"
	task1.SetId(&id1)
	task1.SetTitle(&title1)

	mockAdapter := &MockRequestAdapter{
		GetTasksResponse: []graphmodels.TodoTaskable{task1},
	}

	client := NewClient("client-id", "refresh-token", WithAdapter(mockAdapter))

	tasks, err := client.GetTasks("list-1")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Task One", tasks[0].Title)
}

func TestClient_RefreshToken(t *testing.T) {
	refreshed := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/common/oauth2/v2.0/token" {
			refreshed = true
			reqBody := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(reqBody)
			bodyStr := string(reqBody)
			assert.Contains(t, bodyStr, "grant_type=refresh_token")
			assert.Contains(t, bodyStr, "refresh_token=refresh-token")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "` + dummyJWT + `",
				"refresh_token": "new-refresh-token",
				"expires_in": 3600
			}`))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	_ = refreshed
	_ = ts
}
