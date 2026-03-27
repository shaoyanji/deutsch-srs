package anki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	VersionNum int
	HTTPClient *http.Client
}

type requestEnvelope struct {
	Action  string         `json:"action"`
	Version int            `json:"version"`
	Params  map[string]any `json:"params,omitempty"`
}

type responseEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  any             `json:"error"`
}

type CardInfo struct {
	CardID int64    `json:"cardId"`
	Deck   string   `json:"deckName"`
	Front  string   `json:"question"`
	Back   string   `json:"answer"`
	Tags   []string `json:"tags"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		VersionNum: 6,
		HTTPClient: &http.Client{Timeout: 4 * time.Second},
	}
}

func (c *Client) Health(ctx context.Context) error {
	_, err := c.Version(ctx)
	return err
}

func (c *Client) Version(ctx context.Context) (int, error) {
	var v int
	if err := c.Call(ctx, "version", nil, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (c *Client) DeckNames(ctx context.Context) ([]string, error) {
	var decks []string
	if err := c.Call(ctx, "deckNames", nil, &decks); err != nil {
		return nil, err
	}
	return decks, nil
}

func (c *Client) FindCards(ctx context.Context, query string) ([]int64, error) {
	var ids []int64
	if err := c.Call(ctx, "findCards", map[string]any{"query": query}, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c *Client) CardsInfo(ctx context.Context, ids []int64) ([]CardInfo, error) {
	var cards []CardInfo
	if err := c.Call(ctx, "cardsInfo", map[string]any{"cards": ids}, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func (c *Client) AddNote(ctx context.Context, deck, front, back string, tags []string) (int64, error) {
	params := map[string]any{
		"note": map[string]any{
			"deckName":  deck,
			"modelName": "Basic",
			"fields": map[string]any{
				"Front": front,
				"Back":  back,
			},
			"tags": tags,
		},
	}
	var id int64
	if err := c.Call(ctx, "addNote", params, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) NumReviewsToday(ctx context.Context) (int, error) {
	var n int
	if err := c.Call(ctx, "getNumCardsReviewedToday", nil, &n); err != nil {
		return 0, err
	}
	return n, nil
}

func (c *Client) Call(ctx context.Context, action string, params map[string]any, out any) error {
	reqBody, err := json.Marshal(requestEnvelope{
		Action:  action,
		Version: c.VersionNum,
		Params:  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ankiconnect status %d", resp.StatusCode)
	}

	var env responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if env.Error != nil {
		return fmt.Errorf("ankiconnect error: %v", env.Error)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	return nil
}
