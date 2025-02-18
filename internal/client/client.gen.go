// Package client provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	geth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	geth_core_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/oapi-codegen/runtime"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// PreconfFeeResponse denominated in wei
type PreconfFeeResponse struct {
	BlobGasFee uint64 `json:"blob_gas_fee"`
	GasFee     uint64 `json:"gas_fee"`
}

// ReserveBlockSpaceRequest defines model for ReserveBlockSpaceRequest.
type ReserveBlockSpaceRequest struct {
	BlobCount uint32 `json:"blob_count"`

	// Deposit This is the amount deducted from the user's escrow balance when the user fails to submit a transaction for the allocated blockspace.
	//
	// The deposit is calculated as follows:
	// { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5
	Deposit    geth_hexutil.U256 `json:"deposit"`
	GasLimit   uint64            `json:"gas_limit"`
	TargetSlot uint64            `json:"target_slot"`

	// Tip This is the amount deducted from the user's escrow balance along with `[deposit]` when the user submits a transaction for the allocated blockspace.
	//
	// The tip is calculated as follows:
	// { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5
	Tip geth_hexutil.U256 `json:"tip"`
}

// ReserveBlockSpaceResponse defines model for ReserveBlockSpaceResponse.
type ReserveBlockSpaceResponse struct {
	RequestId openapi_types.UUID `json:"request_id"`

	// Signature An ECDSA signature signed over request body and request id by the gateway
	Signature string `json:"signature"`
}

// SlotInfo defines model for SlotInfo.
type SlotInfo struct {
	BlobsAvailable       uint32  `json:"blobs_available"`
	ConstraintsAvailable *uint32 `json:"constraints_available,omitempty"`
	GasAvailable         uint64  `json:"gas_available"`
	Slot                 uint64  `json:"slot"`
}

// SubmitTransactionRequest defines model for SubmitTransactionRequest.
type SubmitTransactionRequest struct {
	RequestId   openapi_types.UUID          `json:"request_id"`
	Transaction geth_core_types.Transaction `json:"transaction"`
}

// GetFeeParams defines parameters for GetFee.
type GetFeeParams struct {
	// Slot slot to fetch fee for
	Slot uint64 `form:"slot" json:"slot"`
}

// ReserveBlockspaceParams defines parameters for ReserveBlockspace.
type ReserveBlockspaceParams struct {
	// XTaiyiSignature An ECDSA signature from the user over fields of request body
	XTaiyiSignature string `json:"x-taiyi-signature"`
}

// SubmitTransactionParams defines parameters for SubmitTransaction.
type SubmitTransactionParams struct {
	// XTaiyiSignature An ECDSA signature from the user over fields of body.
	XTaiyiSignature string `json:"x-taiyi-signature"`
}

// ReserveBlockspaceJSONRequestBody defines body for ReserveBlockspace for application/json ContentType.
type ReserveBlockspaceJSONRequestBody = ReserveBlockSpaceRequest

// SubmitTransactionJSONRequestBody defines body for SubmitTransaction for application/json ContentType.
type SubmitTransactionJSONRequestBody = SubmitTransactionRequest

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetSlots request
	GetSlots(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetFee request
	GetFee(ctx context.Context, params *GetFeeParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ReserveBlockspaceWithBody request with any body
	ReserveBlockspaceWithBody(ctx context.Context, params *ReserveBlockspaceParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	ReserveBlockspace(ctx context.Context, params *ReserveBlockspaceParams, body ReserveBlockspaceJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// SubmitTransactionWithBody request with any body
	SubmitTransactionWithBody(ctx context.Context, params *SubmitTransactionParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	SubmitTransaction(ctx context.Context, params *SubmitTransactionParams, body SubmitTransactionJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetSlots(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSlotsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetFee(ctx context.Context, params *GetFeeParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetFeeRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ReserveBlockspaceWithBody(ctx context.Context, params *ReserveBlockspaceParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewReserveBlockspaceRequestWithBody(c.Server, params, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ReserveBlockspace(ctx context.Context, params *ReserveBlockspaceParams, body ReserveBlockspaceJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewReserveBlockspaceRequest(c.Server, params, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) SubmitTransactionWithBody(ctx context.Context, params *SubmitTransactionParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewSubmitTransactionRequestWithBody(c.Server, params, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) SubmitTransaction(ctx context.Context, params *SubmitTransactionParams, body SubmitTransactionJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewSubmitTransactionRequest(c.Server, params, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetSlotsRequest generates requests for GetSlots
func NewGetSlotsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/commitments/v1/epoch_info")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetFeeRequest generates requests for GetFee
func NewGetFeeRequest(server string, params *GetFeeParams) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/commitments/v1/preconf_fee")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "slot", runtime.ParamLocationQuery, params.Slot); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewReserveBlockspaceRequest calls the generic ReserveBlockspace builder with application/json body
func NewReserveBlockspaceRequest(server string, params *ReserveBlockspaceParams, body ReserveBlockspaceJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewReserveBlockspaceRequestWithBody(server, params, "application/json", bodyReader)
}

// NewReserveBlockspaceRequestWithBody generates requests for ReserveBlockspace with any type of body
func NewReserveBlockspaceRequestWithBody(server string, params *ReserveBlockspaceParams, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/commitments/v1/reserve_blockspace")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	if params != nil {

		var headerParam0 string

		headerParam0, err = runtime.StyleParamWithLocation("simple", false, "x-taiyi-signature", runtime.ParamLocationHeader, params.XTaiyiSignature)
		if err != nil {
			return nil, err
		}

		req.Header.Set("x-taiyi-signature", headerParam0)

	}

	return req, nil
}

// NewSubmitTransactionRequest calls the generic SubmitTransaction builder with application/json body
func NewSubmitTransactionRequest(server string, params *SubmitTransactionParams, body SubmitTransactionJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewSubmitTransactionRequestWithBody(server, params, "application/json", bodyReader)
}

// NewSubmitTransactionRequestWithBody generates requests for SubmitTransaction with any type of body
func NewSubmitTransactionRequestWithBody(server string, params *SubmitTransactionParams, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/commitments/v1/submit_transaction")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	if params != nil {

		var headerParam0 string

		headerParam0, err = runtime.StyleParamWithLocation("simple", false, "x-taiyi-signature", runtime.ParamLocationHeader, params.XTaiyiSignature)
		if err != nil {
			return nil, err
		}

		req.Header.Set("x-taiyi-signature", headerParam0)

	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetSlotsWithResponse request
	GetSlotsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSlotsResponse, error)

	// GetFeeWithResponse request
	GetFeeWithResponse(ctx context.Context, params *GetFeeParams, reqEditors ...RequestEditorFn) (*GetFeeResponse, error)

	// ReserveBlockspaceWithBodyWithResponse request with any body
	ReserveBlockspaceWithBodyWithResponse(ctx context.Context, params *ReserveBlockspaceParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*ReserveBlockspaceResponse, error)

	ReserveBlockspaceWithResponse(ctx context.Context, params *ReserveBlockspaceParams, body ReserveBlockspaceJSONRequestBody, reqEditors ...RequestEditorFn) (*ReserveBlockspaceResponse, error)

	// SubmitTransactionWithBodyWithResponse request with any body
	SubmitTransactionWithBodyWithResponse(ctx context.Context, params *SubmitTransactionParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*SubmitTransactionResponse, error)

	SubmitTransactionWithResponse(ctx context.Context, params *SubmitTransactionParams, body SubmitTransactionJSONRequestBody, reqEditors ...RequestEditorFn) (*SubmitTransactionResponse, error)
}

type GetSlotsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *[]SlotInfo
}

// Status returns HTTPResponse.Status
func (r GetSlotsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetSlotsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetFeeResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *PreconfFeeResponse
	JSON500      *struct {
		// Code Either specific error code in case of invalid request or http status code
		Code *float32 `json:"code,omitempty"`

		// Message Message describing error
		Message *string `json:"message,omitempty"`
	}
}

// Status returns HTTPResponse.Status
func (r GetFeeResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetFeeResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ReserveBlockspaceResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *ReserveBlockSpaceResponse
	JSON400      *struct {
		// Code Either specific error code in case of invalid request or http status code
		Code float32 `json:"code"`

		// Message Message describing error
		Message string `json:"message"`
	}
	JSON500 *struct {
		// Code Either specific error code in case of invalid request or http status code
		Code *float32 `json:"code,omitempty"`

		// Message Message describing error
		Message *string `json:"message,omitempty"`
	}
}

// Status returns HTTPResponse.Status
func (r ReserveBlockspaceResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ReserveBlockspaceResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type SubmitTransactionResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *string
	JSON400      *struct {
		// Code Either specific error code in case of invalid request or http status code
		Code float32 `json:"code"`

		// Message Message describing error
		Message string `json:"message"`
	}
	JSON500 *struct {
		// Code Either specific error code in case of invalid request or http status code
		Code *float32 `json:"code,omitempty"`

		// Message Message describing error
		Message *string `json:"message,omitempty"`
	}
}

// Status returns HTTPResponse.Status
func (r SubmitTransactionResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r SubmitTransactionResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetSlotsWithResponse request returning *GetSlotsResponse
func (c *ClientWithResponses) GetSlotsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSlotsResponse, error) {
	rsp, err := c.GetSlots(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSlotsResponse(rsp)
}

// GetFeeWithResponse request returning *GetFeeResponse
func (c *ClientWithResponses) GetFeeWithResponse(ctx context.Context, params *GetFeeParams, reqEditors ...RequestEditorFn) (*GetFeeResponse, error) {
	rsp, err := c.GetFee(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetFeeResponse(rsp)
}

// ReserveBlockspaceWithBodyWithResponse request with arbitrary body returning *ReserveBlockspaceResponse
func (c *ClientWithResponses) ReserveBlockspaceWithBodyWithResponse(ctx context.Context, params *ReserveBlockspaceParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*ReserveBlockspaceResponse, error) {
	rsp, err := c.ReserveBlockspaceWithBody(ctx, params, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseReserveBlockspaceResponse(rsp)
}

func (c *ClientWithResponses) ReserveBlockspaceWithResponse(ctx context.Context, params *ReserveBlockspaceParams, body ReserveBlockspaceJSONRequestBody, reqEditors ...RequestEditorFn) (*ReserveBlockspaceResponse, error) {
	rsp, err := c.ReserveBlockspace(ctx, params, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseReserveBlockspaceResponse(rsp)
}

// SubmitTransactionWithBodyWithResponse request with arbitrary body returning *SubmitTransactionResponse
func (c *ClientWithResponses) SubmitTransactionWithBodyWithResponse(ctx context.Context, params *SubmitTransactionParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*SubmitTransactionResponse, error) {
	rsp, err := c.SubmitTransactionWithBody(ctx, params, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseSubmitTransactionResponse(rsp)
}

func (c *ClientWithResponses) SubmitTransactionWithResponse(ctx context.Context, params *SubmitTransactionParams, body SubmitTransactionJSONRequestBody, reqEditors ...RequestEditorFn) (*SubmitTransactionResponse, error) {
	rsp, err := c.SubmitTransaction(ctx, params, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseSubmitTransactionResponse(rsp)
}

// ParseGetSlotsResponse parses an HTTP response from a GetSlotsWithResponse call
func ParseGetSlotsResponse(rsp *http.Response) (*GetSlotsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSlotsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest []SlotInfo
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetFeeResponse parses an HTTP response from a GetFeeWithResponse call
func ParseGetFeeResponse(rsp *http.Response) (*GetFeeResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetFeeResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest PreconfFeeResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest struct {
			// Code Either specific error code in case of invalid request or http status code
			Code *float32 `json:"code,omitempty"`

			// Message Message describing error
			Message *string `json:"message,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseReserveBlockspaceResponse parses an HTTP response from a ReserveBlockspaceWithResponse call
func ParseReserveBlockspaceResponse(rsp *http.Response) (*ReserveBlockspaceResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &ReserveBlockspaceResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest ReserveBlockSpaceResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest struct {
			// Code Either specific error code in case of invalid request or http status code
			Code float32 `json:"code"`

			// Message Message describing error
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest struct {
			// Code Either specific error code in case of invalid request or http status code
			Code *float32 `json:"code,omitempty"`

			// Message Message describing error
			Message *string `json:"message,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseSubmitTransactionResponse parses an HTTP response from a SubmitTransactionWithResponse call
func ParseSubmitTransactionResponse(rsp *http.Response) (*SubmitTransactionResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &SubmitTransactionResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest string
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest struct {
			// Code Either specific error code in case of invalid request or http status code
			Code float32 `json:"code"`

			// Message Message describing error
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest struct {
			// Code Either specific error code in case of invalid request or http status code
			Code *float32 `json:"code,omitempty"`

			// Message Message describing error
			Message *string `json:"message,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}
