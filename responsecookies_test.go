package traefik_response_cookies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateConfig(t *testing.T) {
	config := CreateConfig()

	if config == nil {
		t.Fatal("CreateConfig returned nil")
	}

	if config.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected default status code %d, got %d",
			http.StatusOK,
			config.StatusCode,
		)
	}

	if config.Cookies == nil {
		t.Fatal("expected Cookies to be initialized")
	}

	if len(config.Cookies) != 0 {
		t.Fatalf(
			"expected no default cookies, got %d",
			len(config.Cookies),
		)
	}
}

func TestMultipleResponseCookies(t *testing.T) {
	maxAge := 0
	backendCalled := false

	backend := http.HandlerFunc(func(
		rw http.ResponseWriter,
		_ *http.Request,
	) {
		backendCalled = true
		rw.WriteHeader(http.StatusInternalServerError)
	})

	config := &Config{
		StatusCode: http.StatusOK,
		Cookies: []CookieConfig{
			{
				Name:     "SESSION_ID",
				Value:    "",
				Path:     "/auth/realm/",
				Expires:  "1970-01-01T00:00:01Z",
				MaxAge:   &maxAge,
				Secure:   true,
				HTTPOnly: true,
				SameSite: "none",
			},
			{
				Name:     "IDENTITY",
				Value:    "",
				Path:     "/auth/realm/",
				Expires:  "1970-01-01T00:00:01Z",
				MaxAge:   &maxAge,
				Secure:   true,
				HTTPOnly: true,
				SameSite: "none",
			},
			{
				Name:    "SESSION",
				Value:   "",
				Path:    "/auth/realm/",
				Expires: "1970-01-01T00:00:01Z",
				MaxAge:  &maxAge,
				Secure:  true,
			},
			{
				Name:     "IDENTIFICATION",
				Value:    "",
				Path:     "/",
				Domain:   ".example.com",
				Expires:  "1970-01-01T00:00:01Z",
				MaxAge:   &maxAge,
				Secure:   true,
				HTTPOnly: true,
				SameSite: "lax",
			},
		},
	}

	handler, err := New(
		context.Background(),
		backend,
		config,
		"test-response-cookies",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/logout", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if backendCalled {
		t.Fatal("expected backend not to be called")
	}

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status code %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	if response.ContentLength != 0 {
		t.Fatalf(
			"expected Content-Length 0, got %d",
			response.ContentLength,
		)
	}

	rawCookies := response.Header.Values("Set-Cookie")

	if len(rawCookies) != len(config.Cookies) {
		t.Fatalf(
			"expected %d Set-Cookie headers, got %d: %v",
			len(config.Cookies),
			len(rawCookies),
			rawCookies,
		)
	}

	for _, rawCookie := range rawCookies {
		if !strings.Contains(rawCookie, "Max-Age=0") {
			t.Errorf(
				"expected Set-Cookie header to contain Max-Age=0, got %q",
				rawCookie,
			)
		}
	}

	assertCookie(t, response.Cookies(), cookieExpectation{
		Name:     "SESSION_ID",
		Path:     "/auth/realm/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})

	assertCookie(t, response.Cookies(), cookieExpectation{
		Name:     "IDENTITY",
		Path:     "/auth/realm/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})

	assertCookie(t, response.Cookies(), cookieExpectation{
		Name:     "SESSION",
		Path:     "/auth/realm/",
		Secure:   true,
		HTTPOnly: false,
		SameSite: http.SameSite(0),
		MaxAge:   -1,
	})

	assertCookie(t, response.Cookies(), cookieExpectation{
		Name:     "IDENTIFICATION",
		Path:     "/",
		Domain:   "example.com",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func TestDefaultStatusCode(t *testing.T) {
	config := &Config{
		StatusCode: 0,
		Cookies: []CookieConfig{
			{
				Name:  "test",
				Value: "value",
				Path:  "/",
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected default status code %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}
}

func TestConfiguredStatusCode(t *testing.T) {
	config := &Config{
		StatusCode: http.StatusNoContent,
		Cookies: []CookieConfig{
			{
				Name:  "test",
				Value: "value",
				Path:  "/",
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status code %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}
}

func TestPositiveMaxAge(t *testing.T) {
	maxAge := 3600

	config := &Config{
		Cookies: []CookieConfig{
			{
				Name:   "persistent",
				Value:  "value",
				Path:   "/",
				MaxAge: &maxAge,
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	rawCookie := recorder.Header().Get("Set-Cookie")

	if !strings.Contains(rawCookie, "Max-Age=3600") {
		t.Fatalf(
			"expected Max-Age=3600, got %q",
			rawCookie,
		)
	}
}

func TestMaxAgeOmitted(t *testing.T) {
	config := &Config{
		Cookies: []CookieConfig{
			{
				Name:  "session",
				Value: "value",
				Path:  "/",
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	rawCookie := recorder.Header().Get("Set-Cookie")

	if strings.Contains(rawCookie, "Max-Age") {
		t.Fatalf(
			"expected Max-Age to be omitted, got %q",
			rawCookie,
		)
	}
}

func TestExpires(t *testing.T) {
	config := &Config{
		Cookies: []CookieConfig{
			{
				Name:    "expired",
				Value:   "",
				Path:    "/",
				Expires: "1970-01-01T00:00:01Z",
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	cookies := response.Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}

	expected := time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)

	if !cookies[0].Expires.Equal(expected) {
		t.Fatalf(
			"expected expiration %s, got %s",
			expected,
			cookies[0].Expires,
		)
	}
}

func TestNewValidationErrors(t *testing.T) {
	negativeMaxAge := -1

	tests := []struct {
		name          string
		config        *Config
		expectedError string
	}{
		{
			name:          "nil configuration",
			config:        nil,
			expectedError: "configuration cannot be nil",
		},
		{
			name: "status code below range",
			config: &Config{
				StatusCode: 199,
				Cookies: []CookieConfig{
					{Name: "test"},
				},
			},
			expectedError: "statusCode must be between 200 and 599",
		},
		{
			name: "status code above range",
			config: &Config{
				StatusCode: 600,
				Cookies: []CookieConfig{
					{Name: "test"},
				},
			},
			expectedError: "statusCode must be between 200 and 599",
		},
		{
			name: "empty cookie list",
			config: &Config{
				StatusCode: http.StatusOK,
				Cookies:    []CookieConfig{},
			},
			expectedError: "at least one cookie must be configured",
		},
		{
			name: "empty cookie name",
			config: &Config{
				Cookies: []CookieConfig{
					{Name: "   "},
				},
			},
			expectedError: "cookie name cannot be empty",
		},
		{
			name: "invalid cookie name",
			config: &Config{
				Cookies: []CookieConfig{
					{Name: "invalid cookie"},
				},
			},
			expectedError: "is invalid",
		},
		{
			name: "invalid expiration",
			config: &Config{
				Cookies: []CookieConfig{
					{
						Name:    "test",
						Expires: "not-a-date",
					},
				},
			},
			expectedError: "invalid expires value",
		},
		{
			name: "negative max age",
			config: &Config{
				Cookies: []CookieConfig{
					{
						Name:   "test",
						MaxAge: &negativeMaxAge,
					},
				},
			},
			expectedError: "maxAge -1",
		},
		{
			name: "invalid same site",
			config: &Config{
				Cookies: []CookieConfig{
					{
						Name:     "test",
						SameSite: "invalid",
					},
				},
			},
			expectedError: "unsupported sameSite",
		},
		{
			name: "same site none without secure",
			config: &Config{
				Cookies: []CookieConfig{
					{
						Name:     "test",
						SameSite: "none",
						Secure:   false,
					},
				},
			},
			expectedError: "SameSite=None but Secure is false",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(
				context.Background(),
				http.NotFoundHandler(),
				test.config,
				"test",
			)

			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			if !strings.Contains(err.Error(), test.expectedError) {
				t.Fatalf(
					"expected error containing %q, got %q",
					test.expectedError,
					err.Error(),
				)
			}
		})
	}
}

func TestSameSiteIsCaseInsensitive(t *testing.T) {
	config := &Config{
		Cookies: []CookieConfig{
			{
				Name:     "test",
				Value:    "value",
				Secure:   true,
				SameSite: "NoNe",
			},
		},
	}

	handler, err := New(
		context.Background(),
		http.NotFoundHandler(),
		config,
		"test",
	)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	rawCookie := recorder.Header().Get("Set-Cookie")

	if !strings.Contains(rawCookie, "SameSite=None") {
		t.Fatalf(
			"expected SameSite=None, got %q",
			rawCookie,
		)
	}
}

type cookieExpectation struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
	MaxAge   int
}

func assertCookie(
	t *testing.T,
	cookies []*http.Cookie,
	expected cookieExpectation,
) {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name != expected.Name {
			continue
		}

		if cookie.Path != expected.Path {
			t.Errorf(
				"cookie %q: expected path %q, got %q",
				expected.Name,
				expected.Path,
				cookie.Path,
			)
		}

		if cookie.Domain != expected.Domain {
			t.Errorf(
				"cookie %q: expected domain %q, got %q",
				expected.Name,
				expected.Domain,
				cookie.Domain,
			)
		}

		if cookie.Secure != expected.Secure {
			t.Errorf(
				"cookie %q: expected Secure=%t, got %t",
				expected.Name,
				expected.Secure,
				cookie.Secure,
			)
		}

		if cookie.HttpOnly != expected.HTTPOnly {
			t.Errorf(
				"cookie %q: expected HttpOnly=%t, got %t",
				expected.Name,
				expected.HTTPOnly,
				cookie.HttpOnly,
			)
		}

		if cookie.SameSite != expected.SameSite {
			t.Errorf(
				"cookie %q: expected SameSite=%d, got %d",
				expected.Name,
				expected.SameSite,
				cookie.SameSite,
			)
		}

		if cookie.MaxAge != expected.MaxAge {
			t.Errorf(
				"cookie %q: expected MaxAge=%d, got %d",
				expected.Name,
				expected.MaxAge,
				cookie.MaxAge,
			)
		}

		return
	}

	t.Errorf("cookie %q was not found", expected.Name)
}
