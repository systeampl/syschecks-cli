package config

import "fmt"

// Flags is the subset of CLI global flags config resolution needs. It exists
// to avoid an import cycle between internal/config and internal/cli — the
// cli package builds a Flags from its GlobalFlags before calling Resolve.
type Flags struct {
	Context string
	Org     string
	APIURL  string
	Token   string
}

// Resolved is the fully resolved connection configuration for a single CLI
// invocation.
type Resolved struct {
	APIURL      string
	Token       string
	Org         string
	ContextName string
}

// Resolve applies precedence rules across flags, the active context, env
// vars, and the on-disk token file to produce a Resolved config.
//
// Precedence:
//   - token:   SYSCHECKS_TOKEN env -> fl.Token -> context token file
//   - api_url: fl.APIURL -> context.APIURL -> SYSCHECKS_API_URL env (env only
//     used when the context has none)
//   - org:     fl.Org -> context.Organization
//
// An empty api_url after all of the above is a config error.
func Resolve(f *File, fl Flags, env func(string) string) (Resolved, error) {
	ctxName := fl.Context
	if ctxName == "" {
		ctxName = f.CurrentContext
	}
	// A named context that does not exist is a mistake worth reporting: it used
	// to resolve to the zero value, so a typo silently ran the command against
	// whatever the environment supplied, or failed further down with a message
	// about a missing API URL or token that named nothing.
	if ctxName != "" {
		if _, ok := f.Contexts[ctxName]; !ok {
			return Resolved{}, fmt.Errorf("unknown context %q: check `syschecks config get-contexts`", ctxName)
		}
	}
	ctx := f.Contexts[ctxName]

	apiURL := ctx.APIURL
	if fl.APIURL != "" {
		apiURL = fl.APIURL
	} else if e := env("SYSCHECKS_API_URL"); e != "" && ctx.APIURL == "" {
		apiURL = e
	}

	token := ""
	switch {
	case env("SYSCHECKS_TOKEN") != "":
		token = env("SYSCHECKS_TOKEN")
	case fl.Token != "":
		token = fl.Token
	default:
		if ctxName != "" {
			token, _ = ReadToken(Dir(), ctxName)
		}
	}

	org := ctx.Organization
	if fl.Org != "" {
		org = fl.Org
	}

	if apiURL == "" {
		return Resolved{}, fmt.Errorf("no API URL: set --api-url, a context, or SYSCHECKS_API_URL")
	}
	return Resolved{APIURL: apiURL, Token: token, Org: org, ContextName: ctxName}, nil
}
