package ton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client queries the TON blockchain via the Vultisig API proxy.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a TON client. baseURL should be the Vultisig API URL (e.g. "https://api.vultisig.com").
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL + "/ton",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WalletInfo contains basic TON wallet information.
type WalletInfo struct {
	Balance string `json:"balance"`
	Status  string `json:"status"`
	Seqno   int    `json:"seqno"`
}

// GetWalletInfo returns balance, status, and seqno for a TON address.
func (c *Client) GetWalletInfo(ctx context.Context, address string) (*WalletInfo, error) {
	// Get balance and status
	balURL := fmt.Sprintf("%s/v3/wallet?%s", c.baseURL, url.Values{"address": {address}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, balURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch wallet: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wallet API returned %d", resp.StatusCode)
	}

	var walletResp struct {
		Balance string `json:"balance"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&walletResp); err != nil {
		return nil, fmt.Errorf("decode wallet: %w", err)
	}

	// Get seqno - only default to 0 for uninit accounts, propagate errors for active ones
	seqno := 0
	if walletResp.Status != "uninit" {
		s, err := c.getSeqno(ctx, address)
		if err != nil {
			return nil, fmt.Errorf("get seqno for active account: %w", err)
		}
		seqno = s
	}

	return &WalletInfo{
		Balance: walletResp.Balance,
		Status:  walletResp.Status,
		Seqno:   seqno,
	}, nil
}

func (c *Client) getSeqno(ctx context.Context, address string) (int, error) {
	seqURL := fmt.Sprintf("%s/v2/getExtendedAddressInformation?%s", c.baseURL, url.Values{"address": {address}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, seqURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("seqno API returned %d", resp.StatusCode)
	}

	var seqResp struct {
		Result struct {
			AccountState struct {
				Seqno int `json:"seqno"`
			} `json:"account_state"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&seqResp); err != nil {
		return 0, err
	}
	return seqResp.Result.AccountState.Seqno, nil
}

// ValidateAddress does basic validation of a TON address.
func ValidateAddress(addr string) error {
	if len(addr) < 40 {
		return fmt.Errorf("address too short: %q", addr)
	}
	// TON addresses start with EQ or UQ (user-friendly) or 0: (raw)
	if addr[0] != 'E' && addr[0] != 'U' && addr[0] != '0' {
		return fmt.Errorf("invalid TON address prefix: %q", addr)
	}
	return nil
}
