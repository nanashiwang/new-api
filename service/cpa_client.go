package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const cpaManagementResponseLimit = 4 << 20

type CPAAccount struct {
	ID             string `json:"id"`
	AuthIndex      string `json:"auth_index"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Provider       string `json:"provider"`
	Label          string `json:"label"`
	Status         string `json:"status"`
	StatusMessage  string `json:"status_message"`
	Disabled       bool   `json:"disabled"`
	Unavailable    bool   `json:"unavailable"`
	Success        int64  `json:"success"`
	Failed         int64  `json:"failed"`
	Email          string `json:"email"`
	ProjectID      string `json:"project_id"`
	AccountType    string `json:"account_type"`
	Account        string `json:"account"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	LastRefresh    string `json:"last_refresh"`
	NextRetryAfter string `json:"next_retry_after"`
	Priority       int    `json:"priority"`
	Note           string `json:"note"`
	Websockets     *bool  `json:"websockets,omitempty"`
}

type cpaAuthFilesResponse struct {
	Files []CPAAccount `json:"files"`
}

var cpaManagementHTTPClient = &http.Client{
	Timeout: 12 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func FetchCPAAccounts(ctx context.Context, site *model.CPASite) ([]CPAAccount, error) {
	if site == nil {
		return nil, errors.New("cpa site is nil")
	}
	if err := site.Validate(true); err != nil {
		return nil, err
	}
	key, err := site.DecryptManagementKey()
	if err != nil {
		return nil, fmt.Errorf("decrypt management key: %w", err)
	}
	if strings.TrimSpace(key) == "" {
		return nil, model.ErrCPASiteKeyRequired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, site.BaseURL()+"/v0/management/auth-files", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := cpaManagementHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CPA management connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, errors.New("CPA management authentication failed")
		case http.StatusNotFound:
			return nil, errors.New("CPA management API is unavailable")
		default:
			return nil, fmt.Errorf("CPA management API returned HTTP %d", resp.StatusCode)
		}
	}

	limited := io.LimitReader(resp.Body, cpaManagementResponseLimit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read CPA management response: %w", err)
	}
	if len(body) > cpaManagementResponseLimit {
		return nil, errors.New("CPA management response is too large")
	}
	payload := cpaAuthFilesResponse{}
	if err := common.DecodeJson(bytes.NewReader(body), &payload); err != nil {
		return nil, fmt.Errorf("decode CPA management response: %w", err)
	}
	if payload.Files == nil {
		payload.Files = make([]CPAAccount, 0)
	}
	return payload.Files, nil
}

func SyncCPASiteContext(ctx context.Context, site *model.CPASite) error {
	accounts, err := FetchCPAAccounts(ctx, site)
	if err != nil {
		_ = model.PersistCPASiteSyncContext(ctx, site.Id, "", model.CPASiteStatusError, err.Error())
		return err
	}
	raw, err := common.Marshal(accounts)
	if err != nil {
		return err
	}
	return model.PersistCPASiteSyncContext(ctx, site.Id, string(raw), model.CPASiteStatusSynced, "")
}

func DecodeCPACachedAccounts(raw string) []CPAAccount {
	accounts := make([]CPAAccount, 0)
	if strings.TrimSpace(raw) == "" {
		return accounts
	}
	if err := common.UnmarshalJsonStr(raw, &accounts); err != nil {
		return make([]CPAAccount, 0)
	}
	return accounts
}
