package httpapi

import (
	"io"
	"mime"
	"net/http"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
)

var tmdbImageName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}\.(jpg|jpeg|png|webp)$`)
var tmdbImageSize = regexp.MustCompile(`^(w92|w154|w185|w342|w500|w780|w300|w1280|w45|h632|original)$`)

const maxTMDBImageBytes = 10 << 20

// serveTMDBImage uses the configured outbound proxy, without depending on S3.
// Only TMDB image paths are accepted; callers cannot choose the upstream host.
func serveTMDBImage(outbound *http.Client) echo.HandlerFunc {
	client := http.Client{Timeout: 15 * time.Second}
	if outbound != nil {
		client = *outbound
	}
	if client.Timeout == 0 {
		client.Timeout = 15 * time.Second
	}
	// Do not allow the upstream to redirect this public endpoint to other hosts.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.Jar = nil
	return func(c echo.Context) error {
		size, filename := c.Param("size"), c.Param("filename")
		if !tmdbImageSize.MatchString(size) || !tmdbImageName.MatchString(filename) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid image path")
		}
		req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet,
			"https://image.tmdb.org/t/p/"+size+"/"+filename, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid image path")
		}
		for _, header := range []string{"If-None-Match", "If-Modified-Since"} {
			req.Header.Set(header, c.Request().Header.Get(header))
		}
		resp, err := client.Do(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "image provider unavailable")
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "image not found")
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
			return echo.NewHTTPError(http.StatusBadGateway, "image provider unavailable")
		}
		cacheHeaders := func() {
			c.Response().Header().Set("Cache-Control", "public, max-age=86400")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			for _, header := range []string{"ETag", "Last-Modified"} {
				if value := resp.Header.Get(header); value != "" {
					c.Response().Header().Set(header, value)
				}
			}
		}
		if resp.StatusCode == http.StatusNotModified {
			cacheHeaders()
			return c.NoContent(http.StatusNotModified)
		}
		contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		switch contentType {
		case "image/jpeg", "image/png", "image/webp":
		default:
			return echo.NewHTTPError(http.StatusBadGateway, "invalid image response")
		}
		if resp.ContentLength > maxTMDBImageBytes {
			return echo.NewHTTPError(http.StatusBadGateway, "image too large")
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxTMDBImageBytes+1))
		if err != nil || len(body) == 0 || len(body) > maxTMDBImageBytes {
			return echo.NewHTTPError(http.StatusBadGateway, "invalid image response")
		}
		cacheHeaders()
		return c.Blob(http.StatusOK, contentType, body)
	}
}
