package traefik_plugin_rerouter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	url2 "net/url"
	"strings"
)

const version = "v0.0.6"

// Config the plugin configuration.
type Config struct {
	Version string
}

const DefaultReRouterHeaderVersion = "X-ReRouter-Traefik-Middleware-Version"
const DefaultReRouterHeaderDefaultURL = "X-ReRouter-Traefik-Middleware-Version"
const DefaultReRouterHeaderReRoutedURL = "X-ReRouter-Traefik-Middleware-Version"

const MiddleWareName = "ReRouter-Middleware"

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Version: version,
	}
}

type SemanticDomain string

func (s SemanticDomain) URL() (string string, hasLookup bool) {
	switch s {
	case Github:
		return "github.com", true
	case App:
		// an den start von der traefik kette routen
		return "traefik.alexheld.io", true
	default:
		return "", false
	}
}

const (
	Github SemanticDomain = "g"
	App    SemanticDomain = "app"
)

// New creates and returns a plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log.Printf("[INFO]  New %s Middleware instantiated; name=%s\n", MiddleWareName, name)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defaultUrl := req.URL.String()
		op, err := getReWriteOperation(req.URL.String())
		if err != nil {
			_, _ = rw.Write([]byte(err.Error()))
			rw.WriteHeader(http.StatusMisdirectedRequest)
			return
		}
		rewrittenRequest := op(req)

		applyHeaders(req, rewrittenRequest, config, defaultUrl)

		next.ServeHTTP(rw, req)
	}), nil
}

func applyHeaders(req *http.Request, rewrittenRequest *http.Request, config *Config, defaultUrl string) {
	req.Header.Set("HOST", rewrittenRequest.URL.Host)
	req.Header.Set(DefaultReRouterHeaderVersion, config.Version)
	req.Header.Set(DefaultReRouterHeaderDefaultURL, defaultUrl)
	req.Header.Set(DefaultReRouterHeaderReRoutedURL, req.URL.String())
}

func getReWriteOperation(host string) (rewrite func(*http.Request) *http.Request, err error) {
	p := strings.Split(".", host)
	if p[1] == string(Github) {
		// faster -> is not my repo!
		return rewriteThirdPartyGithubURI(p)
	} else if p[2] == "" {
		// 2nd position -> is my repo!
		return rewriteMyGithubURI()
	} else {
		// ignoring for now ...
		return rewriteNoOp()
	}
}

func rewriteNoOp() (func(*http.Request) *http.Request, error) {
	return func(r *http.Request) *http.Request {
		log.Printf("[INFO]  NOOP not rewriting url url=%s\n", r.URL.String())
		return r
	}, nil
}

// rewriteMyGithubURI
// gh.alexheld.io/path   -> github.com/alex-held/path
// 2     1     0  path         2        1        path
func rewriteMyGithubURI() (func(*http.Request) *http.Request, error) {
	return func(r *http.Request) *http.Request {
		path := r.URL.RawPath
		rawUrl := fmt.Sprintf("%s://github.com/alex-held/%s", r.URL.Scheme, path)
		newUrl, err := url2.ParseRequestURI(rawUrl)
		if err != nil {
			log.Printf("[ERRO]  unable to parse the new URL; oldURL=%s, newURL=%s\n", r.URL.String(), rawUrl)
			return nil
		}
		r.URL = newUrl
		return r
	}, nil
}

func rewriteThirdPartyGithubURI(p []string) (func(*http.Request) *http.Request, error) {
	// sub.domain.xx/path
	// gh.someone.tl/path   -> github.com/someone/path
	// 2    1     0  path         2        1        path
	return func(r *http.Request) *http.Request {
		path := r.URL.RawPath
		rawUrl := fmt.Sprintf("%s://github.com/%s/%s", r.URL.Scheme, p[1], path)
		newUrl, err := url2.ParseRequestURI(rawUrl)
		if err != nil {
			log.Printf("[ERRO]  unable to parse the new URL; oldURL=%s, newURL=%s\n", r.URL.String(), rawUrl)
		}
		r.URL = newUrl
		return r
	}, nil
}
