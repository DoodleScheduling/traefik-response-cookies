// Package responsecookies provides a Traefik middleware that returns
// a configurable HTTP response containing multiple Set-Cookie headers.
package traefik_response_cookies

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CookieConfig defines a cookie added to the HTTP response.
type CookieConfig struct {
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Expires  string `json:"expires,omitempty"`
	MaxAge   *int   `json:"maxAge,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
}

// Config defines the middleware configuration.
type Config struct {
	StatusCode int            `json:"statusCode,omitempty"`
	Cookies    []CookieConfig `json:"cookies,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		StatusCode: http.StatusOK,
		Cookies:    []CookieConfig{},
	}
}

// ResponseCookies is a terminal middleware that returns the configured
// response without forwarding the request to the backend.
type ResponseCookies struct {
	statusCode int
	cookies    []*http.Cookie
}

// New creates a new ResponseCookies middleware.
func New(
	_ context.Context,
	_ http.Handler,
	config *Config,
	_ string,
) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	statusCode := config.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	if statusCode < 200 || statusCode > 599 {
		return nil, fmt.Errorf(
			"statusCode must be between 200 and 599, got %d",
			statusCode,
		)
	}

	if len(config.Cookies) == 0 {
		return nil, fmt.Errorf("at least one cookie must be configured")
	}

	cookies := make([]*http.Cookie, 0, len(config.Cookies))

	for index, configuredCookie := range config.Cookies {
		cookie, err := buildCookie(configuredCookie)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid cookie at position %d: %w",
				index,
				err,
			)
		}

		cookies = append(cookies, cookie)
	}

	return &ResponseCookies{
		statusCode: statusCode,
		cookies:    cookies,
	}, nil
}

// ServeHTTP adds every configured Set-Cookie header and terminates the
// request without calling the backend.
//
// This reproduces the following Nginx behavior:
//
//	add_header Set-Cookie "...";
//	return 200;
func (middleware *ResponseCookies) ServeHTTP(
	rw http.ResponseWriter,
	_ *http.Request,
) {
	for _, cookie := range middleware.cookies {
		http.SetCookie(rw, cookie)
	}

	rw.Header().Set("Content-Length", "0")
	rw.WriteHeader(middleware.statusCode)
}

func buildCookie(config CookieConfig) (*http.Cookie, error) {
	config.Name = strings.TrimSpace(config.Name)

	if config.Name == "" {
		return nil, fmt.Errorf("cookie name cannot be empty")
	}

	cookie := &http.Cookie{
		Name:     config.Name,
		Value:    config.Value,
		Path:     config.Path,
		Domain:   config.Domain,
		Secure:   config.Secure,
		HttpOnly: config.HTTPOnly,
	}

	if config.Expires != "" {
		expires, err := time.Parse(time.RFC3339, config.Expires)
		if err != nil {
			return nil, fmt.Errorf(
				"cookie %q has invalid expires value %q: %w",
				config.Name,
				config.Expires,
				err,
			)
		}

		cookie.Expires = expires.UTC()
	}

	if config.MaxAge != nil {
		if *config.MaxAge < 0 {
			return nil, fmt.Errorf(
				"cookie %q has invalid maxAge %d: it cannot be negative",
				config.Name,
				*config.MaxAge,
			)
		}

		if *config.MaxAge == 0 {
			// In net/http, MaxAge < 0 generates the HTTP attribute
			// Max-Age=0, which deletes the cookie immediately.
			cookie.MaxAge = -1
		} else {
			cookie.MaxAge = *config.MaxAge
		}
	}

	sameSite, err := parseSameSite(config.Name, config.SameSite, config.Secure)
	if err != nil {
		return nil, err
	}

	cookie.SameSite = sameSite

	if err := cookie.Valid(); err != nil {
		return nil, fmt.Errorf(
			"cookie %q is invalid: %w",
			config.Name,
			err,
		)
	}

	return cookie, nil
}

func parseSameSite(
	cookieName string,
	value string,
	secure bool,
) (http.SameSite, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return http.SameSiteDefaultMode, nil

	case "lax":
		return http.SameSiteLaxMode, nil

	case "strict":
		return http.SameSiteStrictMode, nil

	case "none":
		if !secure {
			return http.SameSiteDefaultMode, fmt.Errorf(
				"cookie %q uses SameSite=None but Secure is false",
				cookieName,
			)
		}

		return http.SameSiteNoneMode, nil

	default:
		return http.SameSiteDefaultMode, fmt.Errorf(
			"cookie %q has unsupported sameSite value %q",
			cookieName,
			value,
		)
	}
}
