package connections

import (
	"context"
)

// demoSecret is only for local Compose; it is not a real vendor credential.
const demoSecret = "demo-secret"

// SeedDemo creates verified fakevendor connections when the workspace has none.
// It is idempotent across API restarts on the same database volume.
func SeedDemo(ctx context.Context, svc *Service, workspaceID string) error {
	existing, err := svc.List(ctx, workspaceID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	demos := []string{"Acme Billing (demo)", "Staging vendor (demo)"}
	for _, name := range demos {
		conn, err := svc.Create(ctx, workspaceID, CreateInput{
			Kind:   "fakevendor",
			Name:   name,
			Secret: demoSecret,
		})
		if err != nil {
			return err
		}
		if _, err := svc.Verify(ctx, workspaceID, conn.ID); err != nil {
			return err
		}
	}
	return nil
}
