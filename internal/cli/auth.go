package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/config"
	"github.com/systeampl/syschecks-go/models"
)

// newAuthCmd groups the PAT lifecycle: login, logout, whoami.
func newAuthCmd() *cobra.Command {
	c := &cobra.Command{Use: "auth", Short: "Authentication"}
	c.AddCommand(newLoginCmd(), newLogoutCmd(), newWhoamiCmd())
	return c
}

func newLoginCmd() *cobra.Command {
	var withToken bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store a PAT for the current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !withToken {
				return clierr.Config("v0.1 supports only --with-token")
			}
			tok := strings.TrimSpace(firstNonEmpty(args))
			if tok == "" {
				line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				tok = strings.TrimSpace(line)
			}
			if tok == "" {
				return clierr.Config("no token provided")
			}

			// Validate with the candidate token before it is ever persisted.
			env, err := cmdEnvWithToken(cmd, tok)
			if err != nil {
				return err
			}
			me, err := env.SDK.Profile.GetProfile(cmd.Context())
			if err != nil {
				return clierr.Config("token rejected: %v", err)
			}

			ctxName, err := currentContextName(cmd)
			if err != nil {
				return err
			}
			if err := config.WriteToken(config.Dir(), ctxName, tok); err != nil {
				return clierr.Config("writing token: %v", err)
			}
			cmd.Printf("Logged in as %s\n", emailOf(me))
			return nil
		},
	}
	cmd.Flags().BoolVar(&withToken, "with-token", false, "read PAT from arg or stdin")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored PAT for the current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctxName, err := currentContextName(cmd)
			if err != nil {
				return err
			}
			path := config.TokenPath(config.Dir(), ctxName)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return clierr.Config("removing token: %v", err)
			}
			cmd.Println("Logged out")
			return nil
		},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the identity of the current PAT",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			me, err := env.SDK.Profile.GetProfile(cmd.Context())
			if err != nil {
				return clierr.Config("fetching profile: %v", err)
			}
			if me.FullName != nil && *me.FullName != "" {
				cmd.Printf("%s (%s)\n", emailOf(me), *me.FullName)
			} else {
				cmd.Println(emailOf(me))
			}
			return nil
		},
	}
}

// currentContextName is the context a login/logout call acts on: the
// --context flag, else the config file's current-context, else "default".
// config.Load returns (nil, err) on a malformed config.yaml (anything other
// than the file not existing), so that error must be propagated rather than
// discarded — dereferencing a nil *config.File would panic instead of
// producing the clean clierr.Config exit the CLI contract requires.
func currentContextName(cmd *cobra.Command) (string, error) {
	if gf := globalsFrom(cmd); gf.Context != "" {
		return gf.Context, nil
	}
	f, err := config.Load(config.Dir())
	if err != nil {
		return "", clierr.Config("loading config: %v", err)
	}
	if f.CurrentContext != "" {
		return f.CurrentContext, nil
	}
	return "default", nil
}

// firstNonEmpty returns the first non-empty string in ss, or "".
func firstNonEmpty(ss []string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// emailOf reads the email out of a profile response.
func emailOf(me *models.UserResponse) string {
	return string(me.Email)
}
