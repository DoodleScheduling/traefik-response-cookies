# Traefik Response Cookies

[![release](https://img.shields.io/github/release/DoodleScheduling/traefik-response-cookies/all.svg)](https://github.com/DoodleScheduling/traefik-response-cookies/releases)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/traefik-response-cookies)](https://goreportcard.com/report/github.com/DoodleScheduling/traefik-response-cookies)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-response-cookies/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-response-cookies)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/traefik-response-cookies/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/traefik-response-cookies?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/traefik-response-cookies.svg)](https://github.com/DoodleScheduling/traefik-response-cookies/blob/master/LICENSE)

A Traefik middleware plugin that returns multiple configurable `Set-Cookie` headers and a configurable HTTP status code.

The plugin is useful for endpoints that must expire or create several cookies and return a response directly from Traefik.

## Behavior

The middleware:

1. Creates one independent `Set-Cookie` response header for every configured cookie.
2. Returns the configured HTTP status code.
3. Sets `Content-Length: 0`.
4. Does not forward the request to the backend service.

Because this is a terminal middleware, it should only be attached to routes that must be handled directly by Traefik.

Example response:

```http
HTTP/1.1 200 OK
Content-Length: 0
Set-Cookie: SESSION_ID=; Path=/auth/realm/; Expires=Thu, 01 Jan 1970 00:00:01 GMT; Max-Age=0; HttpOnly; Secure; SameSite=None
Set-Cookie: IDENTITY=; Path=/auth/realm/; Expires=Thu, 01 Jan 1970 00:00:01 GMT; Max-Age=0; HttpOnly; Secure; SameSite=None
```

## Configuration fields

### Middleware fields

| Key | Description |
| --- | --- |
| `statusCode` | HTTP status code returned by the middleware. Must be between `200` and `599`. Defaults to `200`. |
| `cookies` | List of cookies to add to the response. At least one cookie is required. |

### Cookie fields

| Key | Description |
| --- | --- |
| `name` | Cookie name. Required. |
| `value` | Cookie value. Defaults to an empty string. |
| `path` | Cookie path. Optional. |
| `domain` | Cookie domain. Optional. |
| `expires` | Expiration date in RFC 3339 format, for example `1970-01-01T00:00:01Z`. Optional. |
| `maxAge` | Cookie lifetime in seconds. `0` generates `Max-Age=0`, which expires the cookie immediately. Positive values generate `Max-Age=<seconds>`. Optional. |
| `secure` | Adds the `Secure` attribute when `true`. |
| `httpOnly` | Adds the `HttpOnly` attribute when `true`. |
| `sameSite` | SameSite mode: `default`, `lax`, `strict`, or `none`. `SameSite=None` requires `secure: true`. Optional. |

## Middleware configuration

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: clear-response-cookies
spec:
  plugin:
    responseCookies:
      statusCode: 200
      cookies:
        - name: SESSION_ID
          value: ""
          expires: "1970-01-01T00:00:01Z"
          maxAge: 0
          path: /auth/realm/
          secure: true
          httpOnly: true
          sameSite: none

        - name: IDENTITY
          value: ""
          expires: "1970-01-01T00:00:01Z"
          maxAge: 0
          path: /auth/realm/
          secure: true
          httpOnly: true
          sameSite: none

        - name: SESSION
          value: ""
          expires: "1970-01-01T00:00:01Z"
          maxAge: 0
          path: /auth/realm/
          secure: true

        - name: IDENTIFICATION
          value: ""
          expires: "1970-01-01T00:00:01Z"
          maxAge: 0
          path: /
          domain: .example.com
          secure: true
          httpOnly: true
          sameSite: lax
```

## HTTPRoute configuration

Attach the middleware using an `ExtensionRef` filter:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: clear-response-cookies
spec:
  hostnames:
    - example.com
  parentRefs:
    - name: external-gateway
      namespace: gateway-system
      sectionName: websecure
  rules:
    - matches:
        - path:
            type: Exact
            value: /clear-cookies
      filters:
        - type: ExtensionRef
          extensionRef:
            group: traefik.io
            kind: Middleware
            name: clear-response-cookies
      backendRefs:
        - group: ""
          kind: Service
          name: example-service
          port: 80
```

The `backendRef` is required by the route configuration, but the middleware terminates the request before the backend is called.

## IngressRoute configuration

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: clear-response-cookies
spec:
  entryPoints:
    - websecure
  routes:
    - kind: Rule
      match: Host(`example.com`) && Path(`/clear-cookies`)
      middlewares:
        - name: clear-response-cookies
      services:
        - name: example-service
          port: 80
```

## Static configuration

Enable the plugin in the Traefik static configuration:

```yaml
experimental:
  plugins:
    responseCookies:
      moduleName: github.com/DoodleScheduling/traefik-response-cookies
      version: v0.0.1
```

The key used under `experimental.plugins` must match the key used under `spec.plugin`:

```yaml
responseCookies:
```

## Helm configuration

When installing Traefik with Helm:

```yaml
experimental:
  plugins:
    responseCookies:
      moduleName: github.com/DoodleScheduling/traefik-response-cookies
      version: v0.0.1
```

## Max-Age behavior

The plugin configuration follows the HTTP header representation:

```yaml
maxAge: 0
```

This generates:

```http
Max-Age=0
```

Internally, Go requires a negative `http.Cookie.MaxAge` value to serialize `Max-Age=0`. The plugin performs this conversion automatically.

If `maxAge` is omitted, the `Max-Age` attribute is not included. Positive values are rendered directly:

```yaml
maxAge: 3600
```

```http
Max-Age=3600
```

## Development

Run the test suite:

```bash
go test ./...
```

Run formatting, static analysis, tests, race detection, and coverage through the project Makefile:

```bash
make test
```

## Plugin catalog metadata

The `.traefik.yml` metadata must use the same module name:

```yaml
displayName: Response Cookies
iconPath: .assets/icon.png
import: github.com/DoodleScheduling/traefik-response-cookies
summary: Middleware that returns multiple configurable Set-Cookie headers with a configurable HTTP status code.
type: middleware
testData:
  statusCode: 200
  cookies:
    - name: SESSION_ID
      value: ""
      expires: "1970-01-01T00:00:01Z"
      maxAge: 0
      path: /auth/realm/
      secure: true
      httpOnly: true
      sameSite: none

    - name: IDENTIFICATION
      value: ""
      expires: "1970-01-01T00:00:01Z"
      maxAge: 0
      path: /
      domain: .example.com
      secure: true
      httpOnly: true
      sameSite: lax
```

## License

See [LICENSE](LICENSE).
