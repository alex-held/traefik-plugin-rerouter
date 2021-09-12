package traefik_plugin_rerouter

import (
	"context"
	"fmt"
	"net/http"
	url2 "net/url"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const version = "v0.0.1"

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
	zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.NoColor = false
		w.TimeFormat = "2006-01-02T15:04:05"
	})

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
	log.Info().Str("name", name).Msgf("New %s instantiated", MiddleWareName)

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

/*
var stackSize int

type DomainStack struct {
	parts []string
	top   int
}

func NewDomainStackFromArray(parts []string) (s *DomainStack) {
	rev := make([]string, len(parts))
	for i, p := range parts {
		rev[i-1] = p
	}
	s = &DomainStack{
		parts: rev,
		top:   len(rev) - 1,
	}
	return s
}

func (s *DomainStack) Push(part string) {
	if s.top == stackSize-1 {
		fmt.Println("Stack Overflow, cannot insert more values")
		return
	}

	s.parts = append(s.parts, part)
	s.top++
	log.Trace().Msgf("pushed value %v at position %d", part, s.top)
	return
}

func (s *DomainStack) Pop() (part string, err error) {

	if s.top == -1 {
		err = fmt.Errorf("stack is empty, but pop gets still called")
		log.Error().Err(err).Send()
		return "", err
	}

	popped := s.parts[s.top]
	s.parts = s.parts[:s.top]
	s.top--

	log.Trace().Msgf("popped value %v at position %d", part, s.top)
	return popped, nil
}


*/
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
		log.Info().
			Str("url", r.URL.String()).
			Msgf("NOOP not rewriting url.")
		return r
	}, nil
}
g
// rewriteMyGithubURI
// gh.alexheld.io/path   -> github.com/alex-held/path
// 2     1     0  path         2        1        path
func rewriteMyGithubURI() (func(*http.Request) *http.Request, error) {
	return func(r *http.Request) *http.Request {
		path := r.URL.RawPath
		rawUrl := fmt.Sprintf("%s://github.com/alex-held/%s", r.URL.Scheme, path)
		newUrl, err := url2.ParseRequestURI(rawUrl)
		if err != nil {
			log.Error().Err(err).
				Str("oldUrl", r.URL.String()).
				Str("newUrl", rawUrl).
				Msgf("unable to parse the new uri")
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
			log.Error().Err(err).
				Str("oldUrl", r.URL.String()).
				Str("newUrl", rawUrl).
				Msgf("unable to parse the new uri")
		}
		r.URL = newUrl
		return r
	}, nil
}
