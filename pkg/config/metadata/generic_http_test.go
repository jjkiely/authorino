package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mock_auth "github.com/kuadrant/authorino/pkg/auth/mocks"
	"github.com/kuadrant/authorino/pkg/common"

	envoy_auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	. "github.com/golang/mock/gomock"

	"gotest.tools/assert"
)

const (
	extHttpServiceHost string = "127.0.0.1:9005"
)

func TestGenericHttpCallWithGET(t *testing.T) {
	extHttpMetadataServer := mock_auth.NewHttpServerMock(extHttpServiceHost, map[string]mock_auth.HttpServerMockResponseFunc{
		"/metadata": func() mock_auth.HttpServerMockResponse {
			return mock_auth.HttpServerMockResponse{Status: 200, Body: `{"foo":"bar"}`}
		},
	})
	defer extHttpMetadataServer.Close()

	ctx := context.TODO()
	ctrl := NewController(t)
	defer ctrl.Finish()

	endpoint := "http://" + extHttpServiceHost + "/metadata"

	pipelineMock := mock_auth.NewMockAuthPipeline(ctrl)
	pipelineMock.EXPECT().GetAuthorizationJSON().Return(genericHttpAuthDataMock())

	sharedCredsMock := mock_auth.NewMockAuthCredentials(ctrl)
	httpRequestMock, _ := http.NewRequest("GET", endpoint, nil)
	sharedCredsMock.EXPECT().BuildRequestWithCredentials(ctx, endpoint, "GET", "secret", nil).Return(httpRequestMock, nil)

	metadata := &GenericHttp{
		Endpoint:        endpoint,
		Method:          "GET",
		SharedSecret:    "secret",
		AuthCredentials: sharedCredsMock,
	}

	obj, err := metadata.Call(pipelineMock, ctx)

	assert.NilError(t, err)

	objJSON := obj.(map[string]interface{})
	assert.Equal(t, objJSON["foo"], "bar")
}

func TestGenericHttpCallWithPOST(t *testing.T) {
	extHttpMetadataServer := mock_auth.NewHttpServerMock(extHttpServiceHost, map[string]mock_auth.HttpServerMockResponseFunc{
		"/metadata": func() mock_auth.HttpServerMockResponse {
			return mock_auth.HttpServerMockResponse{Status: 200, Body: `{"foo":"bar"}`}
		},
	})
	defer extHttpMetadataServer.Close()

	ctx := context.TODO()
	ctrl := NewController(t)
	defer ctrl.Finish()

	endpoint := "http://" + extHttpServiceHost + "/metadata"

	pipelineMock := mock_auth.NewMockAuthPipeline(ctrl)
	pipelineMock.EXPECT().GetAuthorizationJSON().Return(genericHttpAuthDataMock())

	sharedCredsMock := mock_auth.NewMockAuthCredentials(ctrl)
	requestBody := bytes.NewBuffer([]byte("user=mock"))
	httpRequestMock, _ := http.NewRequest("POST", endpoint, requestBody)
	sharedCredsMock.EXPECT().BuildRequestWithCredentials(ctx, endpoint, "POST", "secret", requestBody).Return(httpRequestMock, nil)

	metadata := &GenericHttp{
		Endpoint:        endpoint,
		Method:          "POST",
		Parameters:      []common.JSONProperty{{Name: "user", Value: common.JSONValue{Pattern: "auth.identity.user"}}},
		ContentType:     "application/x-www-form-urlencoded",
		SharedSecret:    "secret",
		AuthCredentials: sharedCredsMock,
	}

	obj, err := metadata.Call(pipelineMock, ctx)

	assert.NilError(t, err)

	objJSON := obj.(map[string]interface{})
	assert.Equal(t, objJSON["foo"], "bar")
}

func TestGenericHttpCallWithURLPlaceholders(t *testing.T) {
	extHttpMetadataServer := mock_auth.NewHttpServerMock(extHttpServiceHost, map[string]mock_auth.HttpServerMockResponseFunc{
		"/metadata?p=some-origin": func() mock_auth.HttpServerMockResponse {
			return mock_auth.HttpServerMockResponse{Status: 200, Body: `{"foo":"bar"}`}
		},
	})
	defer extHttpMetadataServer.Close()

	ctx := context.TODO()
	ctrl := NewController(t)
	defer ctrl.Finish()

	endpointWithPlaceholders := "http://" + extHttpServiceHost + "/metadata?p={context.request.http.headers.x-origin}"
	endpoint := "http://" + extHttpServiceHost + "/metadata?p=some-origin"

	pipelineMock := mock_auth.NewMockAuthPipeline(ctrl)
	pipelineMock.EXPECT().GetAuthorizationJSON().Return(genericHttpAuthDataMock())

	sharedCredsMock := mock_auth.NewMockAuthCredentials(ctrl)
	httpRequestMock, _ := http.NewRequest("GET", endpoint, nil)
	sharedCredsMock.EXPECT().BuildRequestWithCredentials(ctx, endpoint, "GET", "secret", nil).Return(httpRequestMock, nil)

	metadata := &GenericHttp{
		Endpoint:        endpointWithPlaceholders,
		Method:          "GET",
		SharedSecret:    "secret",
		AuthCredentials: sharedCredsMock,
	}

	obj, err := metadata.Call(pipelineMock, ctx)

	assert.NilError(t, err)

	objJSON := obj.(map[string]interface{})
	assert.Equal(t, objJSON["foo"], "bar")
}

func TestGenericHttpCallWithCustomHeaders(t *testing.T) {
	extHttpMetadataServer := mock_auth.NewHttpServerMock(extHttpServiceHost, map[string]mock_auth.HttpServerMockResponseFunc{
		"/metadata": func() mock_auth.HttpServerMockResponse {
			return mock_auth.HttpServerMockResponse{Status: 200, Body: `{"foo":"bar"}`}
		},
	})
	defer extHttpMetadataServer.Close()

	ctx := context.TODO()
	ctrl := NewController(t)
	defer ctrl.Finish()

	endpoint := "http://" + extHttpServiceHost + "/metadata"

	pipelineMock := mock_auth.NewMockAuthPipeline(ctrl)
	pipelineMock.EXPECT().GetAuthorizationJSON().Return(genericHttpAuthDataMock())

	sharedCredsMock := mock_auth.NewMockAuthCredentials(ctrl)
	httpRequestMock, _ := http.NewRequest("GET", endpoint, nil)
	sharedCredsMock.EXPECT().BuildRequestWithCredentials(ctx, endpoint, "GET", "", nil).Return(httpRequestMock, nil)

	metadata := &GenericHttp{
		Endpoint: endpoint,
		Method:   "GET",
		Headers: []common.JSONProperty{
			{Name: "X-Requested-By", Value: common.JSONValue{Static: "authorino"}},
			{Name: "Content-Type", Value: common.JSONValue{Static: "to-be-overwritten"}},
		},
		AuthCredentials: sharedCredsMock,
	}

	_, err := metadata.Call(pipelineMock, ctx)

	assert.NilError(t, err)
	assert.Equal(t, httpRequestMock.Header.Get("X-Requested-By"), "authorino")
	assert.Equal(t, httpRequestMock.Header.Get("Content-Type"), "text/plain")
}

func genericHttpAuthDataMock() string {
	type mockIdentityObject struct {
		User string `json:"user"`
	}

	type authorizationJSON struct {
		Context  *envoy_auth.AttributeContext `json:"context"`
		AuthData map[string]interface{}       `json:"auth"`
	}

	authJSON, _ := json.Marshal(&authorizationJSON{
		Context: &envoy_auth.AttributeContext{
			Request: &envoy_auth.AttributeContext_Request{
				Http: &envoy_auth.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-secret-header": "no-one-knows",
						"x-origin":        "some-origin",
					},
				},
			},
		},
		AuthData: map[string]interface{}{
			"identity": &mockIdentityObject{User: "mock"},
		},
	})

	return string(authJSON)
}
