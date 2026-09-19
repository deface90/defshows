package tmdb

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var (
	imdbIDPattern     = regexp.MustCompile(`^tt[0-9]+$`)
	wikidataIDPattern = regexp.MustCompile(`^Q[1-9][0-9]*$`)
)

// wikipediaURL resolves the exact article via the Wikidata ID supplied by TMDB.
// Empty means no article; nil means a temporary failure (keep the stored URL).
func (p *Provider) wikipediaURL(ctx context.Context, id string) *string {
	empty := ""
	if !wikidataIDPattern.MatchString(id) {
		return &empty
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := url.Values{
		"action": {"wbgetentities"}, "format": {"json"}, "ids": {id},
		"props": {"sitelinks/urls"}, "sitefilter": {"ruwiki|enwiki"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.wikidata.org/w/api.php?"+q.Encode(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "defShows/1.0 (https://github.com/deface90/defshows)")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "Wikipedia link lookup failed", "wikidata_id", id, "error", safeRequestError(err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "Wikipedia link lookup failed", "wikidata_id", id, "status", resp.StatusCode)
		return nil
	}
	var data struct {
		Entities map[string]struct {
			Sitelinks map[string]struct {
				URL string `json:"url"`
			} `json:"sitelinks"`
		} `json:"entities"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		slog.WarnContext(ctx, "Wikipedia link response invalid", "wikidata_id", id)
		return nil
	}
	entity, ok := data.Entities[id]
	if !ok {
		return nil
	}
	for _, lang := range []string{"ru", "en"} {
		link := entity.Sitelinks[lang+"wiki"].URL
		u, err := url.Parse(link)
		if err == nil && u.Scheme == "https" && u.Host == lang+".wikipedia.org" && u.User == nil && u.Path != "" {
			return &link
		}
	}
	return &empty
}
