// Package omdb implements provider.RatingProvider on top of the generated OMDb
// client.
package omdb

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	omdbclient "github.com/deface90/defshows/backend/pkg/clients/omdb"
	"github.com/deface90/defshows/backend/pkg/provider"
)

const defaultBaseURL = "https://www.omdbapi.com"

// Provider is the OMDb-backed RatingProvider.
type Provider struct {
	client *omdbclient.ClientWithResponses
	apiKey string
}

var _ provider.RatingProvider = (*Provider)(nil)

// New creates an OMDb provider. baseURL/httpClient may be empty/nil for defaults.
func New(baseURL, apiKey string, httpClient *http.Client) (*Provider, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	opts := []omdbclient.ClientOption{}
	if httpClient != nil {
		opts = append(opts, omdbclient.WithHTTPClient(httpClient))
	}
	c, err := omdbclient.NewClientWithResponses(baseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("omdb: new client: %w", err)
	}
	return &Provider{client: c, apiKey: apiKey}, nil
}

func (p *Provider) editor(_ context.Context, req *http.Request) error {
	q := req.URL.Query()
	q.Set("apikey", p.apiKey)
	req.URL.RawQuery = q.Encode()
	return nil
}

// Ratings looks up a series by title (and year, if > 0) and returns normalized
// ratings.
func (p *Provider) Ratings(ctx context.Context, title string, year int) ([]provider.Rating, error) {
	params := &omdbclient.GetByTitleParams{T: title, Type: strptr("series")}
	if year > 0 {
		params.Y = strptr(strconv.Itoa(year))
	}
	resp, err := p.client.GetByTitleWithResponse(ctx, params, p.editor)
	if err != nil {
		return nil, fmt.Errorf("omdb: lookup: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("omdb: lookup: unexpected status %d", resp.StatusCode())
	}
	body := resp.JSON200
	if body.Response == nil || !strings.EqualFold(*body.Response, "True") {
		// Not found is not an error; there are simply no ratings.
		return nil, nil
	}

	var ratings []provider.Rating
	if body.Ratings != nil {
		for _, r := range *body.Ratings {
			source := normalizeSource(val(r.Source))
			if source == "" || val(r.Value) == "" {
				continue
			}
			rating := provider.Rating{Source: source, Value: val(r.Value)}
			if source == "imdb" {
				rating.Votes = parseVotes(body.ImdbVotes)
			}
			ratings = append(ratings, rating)
		}
	}
	return ratings, nil
}

func normalizeSource(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "internet movie database":
		return "imdb"
	case "rotten tomatoes":
		return "rotten_tomatoes"
	case "metacritic":
		return "metacritic"
	default:
		return ""
	}
}

func parseVotes(s *string) *int64 {
	if s == nil {
		return nil
	}
	clean := strings.ReplaceAll(*s, ",", "")
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func strptr(s string) *string { return &s }

func val(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
