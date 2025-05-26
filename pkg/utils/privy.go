package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LinkedAccount struct {
	Type             string `json:"type"`
	Address          string `json:"address"`
	VerifiedAt       int64  `json:"verified_at"`
	FirstVerifiedAt  int64  `json:"first_verified_at"`
	LatestVerifiedAt int64  `json:"latest_verified_at"`
}

type PrivyUser struct {
	ID               string          `json:"id"`
	CreatedAt        int64           `json:"created_at"`
	LinkedAccounts   []LinkedAccount `json:"linked_accounts"`
	MFAMethods       []interface{}   `json:"mfa_methods"`
	HasAcceptedTerms bool            `json:"has_accepted_terms"`
	IsGuest          bool            `json:"is_guest"`
}

// Extracts the first email address if present
func (pu *PrivyUser) GetEmail() string {
	for _, acc := range pu.LinkedAccounts {
		if acc.Type == "email" {
			return acc.Address
		}
	}
	return ""
}

func GetPrivyUserByDID(did, appID, appSecret string) (*PrivyUser, error) {
	url := fmt.Sprintf("https://auth.privy.io/api/v1/users/%s", did)

	// Build basic auth header
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", appID, appSecret)))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("privy-app-id", appID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user: %s", string(body))
	}

	var user PrivyUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
