package appservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	apptestutils "github.com/openmeterio/openmeter/openmeter/app/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// recordingHook records every hook invocation for assertion in tests.
type recordingHook struct {
	calls []string
}

func (h *recordingHook) PostCreate(_ context.Context, _ *app.AppBase) error {
	h.calls = append(h.calls, "PostCreate")
	return nil
}

func (h *recordingHook) PreUpdate(_ context.Context, _ *app.AppBase) error {
	h.calls = append(h.calls, "PreUpdate")
	return nil
}

func (h *recordingHook) PostUpdate(_ context.Context, _ *app.AppBase) error {
	h.calls = append(h.calls, "PostUpdate")
	return nil
}

func (h *recordingHook) PreDelete(_ context.Context, _ *app.AppBase) error {
	h.calls = append(h.calls, "PreDelete")
	return nil
}

func (h *recordingHook) PostDelete(_ context.Context, _ *app.AppBase) error {
	h.calls = append(h.calls, "PostDelete")
	return nil
}

var _ models.ServiceHook[app.AppBase] = (*recordingHook)(nil)

// failingHook returns a fixed error from the named hook method.
type failingHook struct {
	models.NoopServiceHook[app.AppBase]
	method string
	err    error
}

func (h *failingHook) PostCreate(_ context.Context, _ *app.AppBase) error {
	if h.method == "PostCreate" {
		return h.err
	}
	return nil
}

func (h *failingHook) PreDelete(_ context.Context, _ *app.AppBase) error {
	if h.method == "PreDelete" {
		return h.err
	}
	return nil
}

func newTestNamespace() string {
	return "ns-" + ulid.Make().String()
}

func TestAppService_CreateAppFiresPostCreateHook(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	hook := &recordingHook{}
	env.AppService.RegisterHooks(hook)

	namespace := newTestNamespace()
	ctx := t.Context()

	_, err := env.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace:   namespace,
		Name:        "Test App",
		Description: "Test",
		Type:        app.AppTypeSandbox,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"PostCreate"}, hook.calls, "PostCreate must fire after CreateApp")
}

func TestAppService_UpdateAppFiresPreUpdateAndPostUpdateHooks(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	namespace := newTestNamespace()
	ctx := t.Context()

	created, err := env.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace:   namespace,
		Name:        "Test App",
		Description: "Test",
		Type:        app.AppTypeSandbox,
	})
	require.NoError(t, err)

	hook := &recordingHook{}
	env.AppService.RegisterHooks(hook)

	_, err = env.AppService.UpdateApp(ctx, app.UpdateAppInput{
		AppID: app.AppID{Namespace: namespace, ID: created.ID},
		Name:  "Updated Name",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"PreUpdate", "PostUpdate"}, hook.calls, "PreUpdate then PostUpdate must fire for UpdateApp")
}

func TestAppService_UninstallAppFiresPreDeleteAndPostDeleteHooks(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	namespace := newTestNamespace()
	ctx := t.Context()

	created, err := env.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace:   namespace,
		Name:        "Test App",
		Description: "Test",
		Type:        app.AppTypeSandbox,
	})
	require.NoError(t, err)

	hook := &recordingHook{}
	env.AppService.RegisterHooks(hook)

	err = env.AppService.UninstallApp(ctx, app.AppID{Namespace: namespace, ID: created.ID})
	require.NoError(t, err)

	assert.Equal(t, []string{"PreDelete", "PostDelete"}, hook.calls, "PreDelete then PostDelete must fire for UninstallApp")
}

func TestAppService_CreateAppRollsBackWhenPostCreateHookFails(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	boom := errors.New("hook boom")
	env.AppService.RegisterHooks(&failingHook{method: "PostCreate", err: boom})

	namespace := newTestNamespace()
	ctx := t.Context()

	created, err := env.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace:   namespace,
		Name:        "Test App",
		Description: "Test",
		Type:        app.AppTypeSandbox,
	})
	require.ErrorIs(t, err, boom, "CreateApp must propagate the hook error")
	assert.Equal(t, app.AppBase{}, created, "returned AppBase must be zero on failure")

	// Verify no row was persisted (transaction rolled back).
	list, err := env.AppService.ListApps(ctx, app.ListAppInput{
		Namespace: namespace,
	})
	require.NoError(t, err)
	assert.Empty(t, list.Items, "no app row must exist after rollback")
}

func TestAppService_UninstallAppRollsBackWhenPreDeleteHookFails(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	namespace := newTestNamespace()
	ctx := t.Context()

	created, err := env.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace:   namespace,
		Name:        "Test App",
		Description: "Test",
		Type:        app.AppTypeSandbox,
	})
	require.NoError(t, err)

	boom := errors.New("pre-delete boom")
	env.AppService.RegisterHooks(&failingHook{method: "PreDelete", err: boom})

	err = env.AppService.UninstallApp(ctx, app.AppID{Namespace: namespace, ID: created.ID})
	require.ErrorIs(t, err, boom, "UninstallApp must propagate the PreDelete hook error")

	// Verify the app row still exists (transaction rolled back).
	gotApp, err := env.AppService.GetApp(ctx, app.AppID{Namespace: namespace, ID: created.ID})
	require.NoError(t, err)
	assert.NotNil(t, gotApp, "app must still exist after rollback")
}
