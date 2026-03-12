package openfga

import (
	"context"
	"fmt"

	fgaclient "github.com/openfga/go-sdk/client"
)

// Tuple represents a single OpenFGA relationship tuple.
type Tuple struct {
	User     string // e.g. "user:uuid" or "organization:uuid"
	Relation string // e.g. "owner", "admin", "platform"
	Object   string // e.g. "organization:uuid", "project:uuid"
}

// WriteTuples writes one or more relationship tuples atomically.
func (c *Client) WriteTuples(ctx context.Context, tuples ...Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	writes := make([]fgaclient.ClientTupleKey, len(tuples))
	for i, t := range tuples {
		writes[i] = fgaclient.ClientTupleKey{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		}
	}
	_, err := c.sdk.Write(ctx).
		Body(fgaclient.ClientWriteRequest{Writes: writes}).
		Execute()
	if err != nil {
		return fmt.Errorf("openfga write: %w", err)
	}
	return nil
}

// DeleteTuples deletes one or more relationship tuples atomically.
func (c *Client) DeleteTuples(ctx context.Context, tuples ...Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	deletes := make([]fgaclient.ClientTupleKeyWithoutCondition, len(tuples))
	for i, t := range tuples {
		deletes[i] = fgaclient.ClientTupleKeyWithoutCondition{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		}
	}
	_, err := c.sdk.Write(ctx).
		Body(fgaclient.ClientWriteRequest{Deletes: deletes}).
		Execute()
	if err != nil {
		return fmt.Errorf("openfga delete: %w", err)
	}
	return nil
}
