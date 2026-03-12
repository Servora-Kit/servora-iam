package openfga

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/Servora-Kit/servora/cmd/svr/internal/envfile"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "openfga",
		Short: "OpenFGA store and model management",
	}
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newModelCmd())
	parent.AddCommand(cmd)
}

// writeEnvVars writes key=value pairs to the env file.
// When envPrefix is non-empty, prefixed variants are also written
// so that Kratos env.NewSource(prefix) can inject them into the config tree.
func writeEnvVars(envFile, envPrefix string, vars map[string]string) error {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := envfile.Upsert(envFile, k, vars[k]); err != nil {
			return err
		}
		if envPrefix != "" {
			if err := envfile.Upsert(envFile, envPrefix+k, vars[k]); err != nil {
				return err
			}
		}
	}
	return nil
}
