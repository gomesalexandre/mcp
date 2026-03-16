package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Hyperliquid REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Hyperliquid API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ExchangeURL returns the full URL for the exchange endpoint.
func (c *Client) ExchangeURL() string {
	return c.baseURL + "/exchange"
}

func (c *Client) postInfo(ctx context.Context, reqBody any, out any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/info", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	err = json.Unmarshal(body, out)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

// SpotBalance represents a single spot token balance.
type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Hold     string `json:"hold"`
	Total    string `json:"total"`
	EntryNtl string `json:"entryNtl"`
}

// SpotClearinghouseState is the response from spotClearinghouseState.
type SpotClearinghouseState struct {
	Balances []SpotBalance `json:"balances"`
}

// MarginSummary holds account value and margin usage.
type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUsd     string `json:"totalRawUsd"`
}

// Leverage holds position leverage details.
type Leverage struct {
	Type   string `json:"type"`
	Value  int    `json:"value"`
	RawUsd string `json:"rawUsd"`
}

// CumFunding holds cumulative funding for a position.
type CumFunding struct {
	AllTime     string `json:"allTime"`
	SinceChange string `json:"sinceChange"`
	SinceOpen   string `json:"sinceOpen"`
}

// Position holds details of a single perpetual position.
type Position struct {
	Coin           string     `json:"coin"`
	EntryPx        string     `json:"entryPx"`
	Leverage       Leverage   `json:"leverage"`
	LiquidationPx  string     `json:"liquidationPx"`
	MarginUsed     string     `json:"marginUsed"`
	MaxLeverage    int        `json:"maxLeverage"`
	PositionValue  string     `json:"positionValue"`
	ReturnOnEquity string     `json:"returnOnEquity"`
	Szi            string     `json:"szi"`
	UnrealizedPnl  string     `json:"unrealizedPnl"`
	CumFunding     CumFunding `json:"cumFunding"`
}

// AssetPosition wraps a Position with its type field.
type AssetPosition struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
}

// ClearinghouseState is the response from clearinghouseState.
type ClearinghouseState struct {
	AssetPositions             []AssetPosition `json:"assetPositions"`
	CrossMaintenanceMarginUsed string          `json:"crossMaintenanceMarginUsed"`
	CrossMarginSummary         MarginSummary   `json:"crossMarginSummary"`
	MarginSummary              MarginSummary   `json:"marginSummary"`
	Withdrawable               string          `json:"withdrawable"`
	Time                       int64           `json:"time"`
}

// OpenOrder represents a single open order with frontend info.
type OpenOrder struct {
	Coin             string `json:"coin"`
	LimitPx          string `json:"limitPx"`
	Oid              int64  `json:"oid"`
	OrderType        string `json:"orderType"`
	OrigSz           string `json:"origSz"`
	ReduceOnly       bool   `json:"reduceOnly"`
	Side             string `json:"side"`
	Sz               string `json:"sz"`
	Timestamp        int64  `json:"timestamp"`
	TriggerCondition string `json:"triggerCondition"`
	TriggerPx        string `json:"triggerPx"`
	IsTrigger        bool   `json:"isTrigger"`
	IsPositionTpsl   bool   `json:"isPositionTpsl"`
	Cloid            string `json:"cloid"`
	Tif              string `json:"tif"`
}

// L2Level represents a single price level in the order book.
type L2Level struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
	N  int    `json:"n"`
}

// L2Book is the response from l2Book.
type L2Book struct {
	Coin   string      `json:"coin"`
	Time   int64       `json:"time"`
	Levels [][]L2Level `json:"levels"`
}

// PerpAsset holds metadata for a single perpetual.
type PerpAsset struct {
	Name         string `json:"name"`
	SzDecimals   int    `json:"szDecimals"`
	MaxLeverage  int    `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated"`
}

// PerpMeta holds metadata for the perpetuals universe.
type PerpMeta struct {
	Universe []PerpAsset `json:"universe"`
}

// SpotToken holds metadata for a single spot token.
type SpotToken struct {
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals"`
	Index       int    `json:"index"`
	TokenId     string `json:"tokenId"`
}

// SpotPair holds metadata for a single spot trading pair.
type SpotPair struct {
	Name        string `json:"name"`
	Tokens      []int  `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// SpotMeta holds metadata for the spot universe.
type SpotMeta struct {
	Tokens   []SpotToken `json:"tokens"`
	Universe []SpotPair  `json:"universe"`
}

// OrderStatusEntry wraps an order with its status.
type OrderStatusEntry struct {
	Order           OpenOrder `json:"order"`
	Status          string    `json:"status"`
	StatusTimestamp int64     `json:"statusTimestamp"`
}

// OrderStatusResponse is the response from orderStatus.
type OrderStatusResponse struct {
	Status string            `json:"status"`
	Order  *OrderStatusEntry `json:"order"`
}

// HistoricalOrder represents a historical order with status.
type HistoricalOrder struct {
	Order           OpenOrder `json:"order"`
	Status          string    `json:"status"`
	StatusTimestamp int64     `json:"statusTimestamp"`
}

// Candle represents an OHLCV candle.
type Candle struct {
	CloseTime int64  `json:"T"`
	Close     string `json:"c"`
	High      string `json:"h"`
	Interval  string `json:"i"`
	Low       string `json:"l"`
	Trades    int    `json:"n"`
	Open      string `json:"o"`
	Coin      string `json:"s"`
	OpenTime  int64  `json:"t"`
	Volume    string `json:"v"`
}

// Fill represents a single trade fill.
type Fill struct {
	ClosedPnl     string `json:"closedPnl"`
	Coin          string `json:"coin"`
	Crossed       bool   `json:"crossed"`
	Dir           string `json:"dir"`
	Fee           string `json:"fee"`
	FeeToken      string `json:"feeToken"`
	Hash          string `json:"hash"`
	Oid           int64  `json:"oid"`
	Px            string `json:"px"`
	Side          string `json:"side"`
	StartPosition string `json:"startPosition"`
	Sz            string `json:"sz"`
	Tid           int64  `json:"tid"`
	Time          int64  `json:"time"`
}

type RateLimit struct {
	CumVlm           string `json:"cumVlm"`
	NRequestsUsed    int    `json:"nRequestsUsed"`
	NRequestsCap     int    `json:"nRequestsCap"`
	NRequestsSurplus int    `json:"nRequestsSurplus"`
}

// GetSpotBalances queries spot token balances for a user address.
func (c *Client) GetSpotBalances(ctx context.Context, user string) (*SpotClearinghouseState, error) {
	reqBody := map[string]string{
		"type": "spotClearinghouseState",
		"user": user,
	}
	var resp SpotClearinghouseState
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get spot balances: %w", err)
	}
	return &resp, nil
}

// GetPerpState queries perpetual positions and margin summary for a user.
func (c *Client) GetPerpState(ctx context.Context, user string) (*ClearinghouseState, error) {
	reqBody := map[string]string{
		"type": "clearinghouseState",
		"user": user,
	}
	var resp ClearinghouseState
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get perp state: %w", err)
	}
	return &resp, nil
}

// GetOpenOrders queries active orders for a user.
func (c *Client) GetOpenOrders(ctx context.Context, user string) ([]OpenOrder, error) {
	reqBody := map[string]string{
		"type": "frontendOpenOrders",
		"user": user,
	}
	var resp []OpenOrder
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get open orders: %w", err)
	}
	return resp, nil
}

// GetAllMids queries mid prices for all coins.
func (c *Client) GetAllMids(ctx context.Context) (map[string]string, error) {
	reqBody := map[string]string{
		"type": "allMids",
	}
	var resp map[string]string
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get all mids: %w", err)
	}
	return resp, nil
}

// GetL2Book queries the order book for a specific coin.
func (c *Client) GetL2Book(ctx context.Context, coin string, nSigFigs int) (*L2Book, error) {
	reqBody := map[string]any{
		"type":     "l2Book",
		"coin":     coin,
		"nSigFigs": nSigFigs,
	}
	var resp L2Book
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get l2 book: %w", err)
	}
	return &resp, nil
}

// GetPerpMeta queries perpetuals universe metadata.
func (c *Client) GetPerpMeta(ctx context.Context) (*PerpMeta, error) {
	reqBody := map[string]string{
		"type": "meta",
	}
	var resp PerpMeta
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get perp meta: %w", err)
	}
	return &resp, nil
}

// GetSpotMeta queries spot token and pair metadata.
func (c *Client) GetSpotMeta(ctx context.Context) (*SpotMeta, error) {
	reqBody := map[string]string{
		"type": "spotMeta",
	}
	var resp SpotMeta
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get spot meta: %w", err)
	}
	return &resp, nil
}

// GetOrderStatusByOid queries order status by order ID.
func (c *Client) GetOrderStatusByOid(ctx context.Context, user string, oid int64) (*OrderStatusResponse, error) {
	reqBody := map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  oid,
	}
	var resp OrderStatusResponse
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get order status by oid: %w", err)
	}
	return &resp, nil
}

// GetOrderStatusByCloid queries order status by client order ID.
func (c *Client) GetOrderStatusByCloid(ctx context.Context, user string, cloid string) (*OrderStatusResponse, error) {
	reqBody := map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  cloid,
	}
	var resp OrderStatusResponse
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get order status by cloid: %w", err)
	}
	return &resp, nil
}

// GetHistoricalOrders queries up to 2000 recent historical orders.
func (c *Client) GetHistoricalOrders(ctx context.Context, user string) ([]HistoricalOrder, error) {
	reqBody := map[string]string{
		"type": "historicalOrders",
		"user": user,
	}
	var resp []HistoricalOrder
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get historical orders: %w", err)
	}
	return resp, nil
}

// GetCandles queries OHLCV candle data for a coin.
func (c *Client) GetCandles(ctx context.Context, coin, interval string, startTime, endTime int64) ([]Candle, error) {
	candleReq := map[string]any{
		"coin":      coin,
		"interval":  interval,
		"startTime": startTime,
	}
	if endTime > 0 {
		candleReq["endTime"] = endTime
	}
	reqBody := map[string]any{
		"type": "candleSnapshot",
		"req":  candleReq,
	}
	var resp []Candle
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get candles: %w", err)
	}
	return resp, nil
}

// GetUserFills queries user's trade fill history.
// If startTime > 0, uses userFillsByTime for time-ranged queries.
func (c *Client) GetUserFills(ctx context.Context, user string, startTime, endTime int64) ([]Fill, error) {
	var reqBody map[string]any
	if startTime > 0 {
		reqBody = map[string]any{
			"type":            "userFillsByTime",
			"user":            user,
			"startTime":       startTime,
			"aggregateByTime": true,
		}
		if endTime > 0 {
			reqBody["endTime"] = endTime
		}
	} else {
		reqBody = map[string]any{
			"type":            "userFills",
			"user":            user,
			"aggregateByTime": true,
		}
	}
	var resp []Fill
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get user fills: %w", err)
	}
	return resp, nil
}

// GetUserRateLimit queries API rate limit status for a user.
func (c *Client) GetUserRateLimit(ctx context.Context, user string) (*RateLimit, error) {
	reqBody := map[string]string{
		"type": "userRateLimit",
		"user": user,
	}
	var resp RateLimit
	err := c.postInfo(ctx, reqBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("get user rate limit: %w", err)
	}
	return &resp, nil
}
