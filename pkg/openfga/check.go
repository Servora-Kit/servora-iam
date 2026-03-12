package openfga

import (
	"context"
	"fmt"

	fgaclient "github.com/openfga/go-sdk/client"
)

// Check returns whether userID has the given relation on objectType:objectID.
func (c *Client) Check(ctx context.Context, userID, relation, objectType, objectID string) (bool, error) {
	resp, err := c.sdk.Check(ctx).
		Body(fgaclient.ClientCheckRequest{
			User:     "user:" + userID,
			Relation: relation,
			Object:   objectType + ":" + objectID,
		}).
		Execute()
	if err != nil {
		return false, fmt.Errorf("openfga check: %w", err)
	}
	return resp.GetAllowed(), nil
}
