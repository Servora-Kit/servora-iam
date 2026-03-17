package openfga

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	fgasdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
	"github.com/openfga/language/pkg/go/transformer"
	"github.com/spf13/cobra"

	"github.com/Servora-Kit/servora/cmd/svr/internal/ux"
)

func newInitCmd() *cobra.Command {
	var (
		apiURL    string
		modelFile string
		storeName string
		envFile   string
		envPrefix string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize OpenFGA store and upload authorization model",
		Long: `Create or reuse an OpenFGA store, upload the authorization model,
and update .env with FGA_STORE_ID and FGA_MODEL_ID.

When --env-prefix is set, prefixed variants (e.g. UNKNOWN_FGA_STORE_ID) are
also written so that Kratos env.NewSource can resolve ${FGA_STORE_ID} in YAML.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.Context(), apiURL, modelFile, storeName, envFile, envPrefix)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", envOrDefault("FGA_API_URL", "http://localhost:18080"), "OpenFGA API URL")
	cmd.Flags().StringVar(&modelFile, "model", "manifests/openfga/model/servora.fga", "path to .fga model file")
	cmd.Flags().StringVar(&storeName, "store", "servora", "OpenFGA store name")
	cmd.Flags().StringVar(&envFile, "env-file", ".env", "path to .env file for output")
	cmd.Flags().StringVar(&envPrefix, "env-prefix", "", "Kratos env source prefix (e.g. UNKNOWN_); writes prefixed vars to .env")

	return cmd
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runInit(ctx context.Context, apiURL, modelFile, storeName, envFile, envPrefix string) error {
	modelFile, err := filepath.Abs(modelFile)
	if err != nil {
		return fmt.Errorf("resolve model path: %w", err)
	}
	dsl, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file: %w", err)
	}

	ux.PrintInfo(fmt.Sprintf("Connecting to OpenFGA at %s...", apiURL))

	sdk, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{ApiUrl: apiURL})
	if err != nil {
		return fmt.Errorf("create OpenFGA client: %w", err)
	}

	storeID, err := findOrCreateStore(ctx, sdk, storeName)
	if err != nil {
		return fmt.Errorf("store setup: %w", err)
	}
	ux.PrintSuccess("Store ready", fmt.Sprintf("(%s: %s)", storeName, storeID))

	modelID, err := writeModel(ctx, apiURL, storeID, string(dsl))
	if err != nil {
		return fmt.Errorf("write model: %w", err)
	}
	ux.PrintSuccess("Model uploaded", fmt.Sprintf("(model_id: %s)", modelID))

	envFile, _ = filepath.Abs(envFile)
	vars := map[string]string{
		"FGA_API_URL":  apiURL,
		"FGA_STORE_ID": storeID,
		"FGA_MODEL_ID": modelID,
	}
	if err := writeEnvVars(envFile, envPrefix, vars); err != nil {
		return err
	}
	ux.PrintSuccess(".env updated", envFile)

	return nil
}

func findOrCreateStore(ctx context.Context, sdk fgaclient.SdkClient, name string) (string, error) {
	resp, err := sdk.ListStores(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("list stores: %w", err)
	}
	for _, s := range resp.Stores {
		if s.GetName() == name {
			return s.GetId(), nil
		}
	}

	ux.PrintInfo(fmt.Sprintf("Creating store '%s'...", name))
	createResp, err := sdk.CreateStore(ctx).Body(fgaclient.ClientCreateStoreRequest{Name: name}).Execute()
	if err != nil {
		return "", fmt.Errorf("create store: %w", err)
	}
	return createResp.GetId(), nil
}

func writeModel(ctx context.Context, apiURL, storeID, dsl string) (string, error) {
	jsonStr, err := transformer.TransformDSLToJSON(dsl)
	if err != nil {
		return "", fmt.Errorf("parse .fga model: %w", err)
	}

	var modelReq fgasdk.WriteAuthorizationModelRequest
	if err := json.Unmarshal([]byte(jsonStr), &modelReq); err != nil {
		return "", fmt.Errorf("unmarshal model JSON: %w", err)
	}

	storeSDK, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:  apiURL,
		StoreId: storeID,
	})
	if err != nil {
		return "", fmt.Errorf("create store-scoped client: %w", err)
	}

	resp, err := storeSDK.WriteAuthorizationModel(ctx).Body(modelReq).Execute()
	if err != nil {
		return "", fmt.Errorf("write authorization model: %w", err)
	}
	return resp.GetAuthorizationModelId(), nil
}
