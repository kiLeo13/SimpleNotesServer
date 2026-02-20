package minhareceita

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"simplenotes/cmd/internal/domain/entity"
)

var (
	ErrNotFound = errors.New("not found")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL:    "https://minhareceita.org/",
		httpClient: &http.Client{},
	}
}

func (c *Client) GetByCNPJ(ctx context.Context, cnpj string) (*entity.Company, error) {
	req := http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			RawPath: c.baseURL + cnpj,
		},
	}
	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minhareceita failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var company companyResponse
	err = json.Unmarshal(body, &company)
	if err != nil {
		return nil, err
	}
	return company.ToDomain(), nil
}
