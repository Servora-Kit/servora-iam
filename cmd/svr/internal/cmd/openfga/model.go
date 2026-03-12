package openfga

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Servora-Kit/servora/cmd/svr/internal/ux"
)

func newModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Authorization model management",
	}
	cmd.AddCommand(newModelApplyCmd())
	return cmd
}

func newModelApplyCmd() *cobra.Command {
	var (
		apiURL    string
		storeID   string
		modelFile string
		envFile   string
		envPrefix string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Upload a new authorization model version to an existing store",
		Long:  "Parse the .fga model file, upload it as a new model version, and update .env with the new FGA_MODEL_ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelApply(cmd.Context(), apiURL, storeID, modelFile, envFile, envPrefix)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", envOrDefault("FGA_API_URL", "http://localhost:8080"), "OpenFGA API URL")
	cmd.Flags().StringVar(&storeID, "store-id", os.Getenv("FGA_STORE_ID"), "OpenFGA store ID")
	cmd.Flags().StringVar(&modelFile, "model", "manifests/openfga/model/servora.fga", "path to .fga model file")
	cmd.Flags().StringVar(&envFile, "env-file", ".env", "path to .env file for output")
	cmd.Flags().StringVar(&envPrefix, "env-prefix", "", "Kratos env source prefix (e.g. UNKNOWN_); writes prefixed vars to .env")

	return cmd
}

func runModelApply(ctx context.Context, apiURL, storeID, modelFile, envFile, envPrefix string) error {
	if storeID == "" {
		return fmt.Errorf("--store-id is required (or set FGA_STORE_ID env)")
	}

	modelFile, err := filepath.Abs(modelFile)
	if err != nil {
		return fmt.Errorf("resolve model path: %w", err)
	}
	dsl, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file: %w", err)
	}

	ux.PrintInfo(fmt.Sprintf("Applying model to store %s...", storeID))

	modelID, err := writeModel(ctx, apiURL, storeID, string(dsl))
	if err != nil {
		return err
	}
	ux.PrintSuccess("Model applied", fmt.Sprintf("(model_id: %s)", modelID))

	envFile, _ = filepath.Abs(envFile)
	if err := writeEnvVars(envFile, envPrefix, map[string]string{"FGA_MODEL_ID": modelID}); err != nil {
		return err
	}
	ux.PrintSuccess(".env updated", envFile)

	return nil
}
