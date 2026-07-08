package openmeter

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPlanAddons_List_NestedPath(t *testing.T) {
	var gotPath, gotRawQuery string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"number":1,"size":10,"total":0}}}`)
	})

	_, err := c.PlanAddons.List(t.Context(), "plan1", PlanAddonListParams{Page: &PageParams{Size: Int(10), Number: Int(1)}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if gotPath != "/openmeter/plans/plan1/addons" {
		t.Fatalf("path = %q, want /openmeter/plans/plan1/addons", gotPath)
	}
	if want := "page%5Bnumber%5D=1&page%5Bsize%5D=10"; gotRawQuery != want {
		t.Fatalf("raw query = %q, want %q", gotRawQuery, want)
	}
}

func TestPlanAddons_Create(t *testing.T) {
	var gotBody CreatePlanAddonRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/openmeter/plans/plan1/addons" {
			t.Errorf("path = %s", r.URL.Path)
		}

		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"pa1","name":"Pro add-on","addon":{"id":"addon1"},"from_plan_phase":"trial","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	pa, err := c.PlanAddons.Create(t.Context(), "plan1", CreatePlanAddonRequest{
		Name:          "Pro add-on",
		Addon:         AddonReference{ID: "addon1"},
		FromPlanPhase: "trial",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if gotBody.Addon.ID != "addon1" || gotBody.FromPlanPhase != "trial" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if pa.ID != "pa1" || pa.Addon.ID != "addon1" {
		t.Fatalf("unexpected plan-addon: %+v", pa)
	}
}

func TestPlanAddons_Get_TwoLevelPath(t *testing.T) {
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"id":"pa1","name":"n","addon":{"id":"addon1"},"from_plan_phase":"trial","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	if _, err := c.PlanAddons.Get(t.Context(), "plan1", "pa1"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if gotPath != "/openmeter/plans/plan1/addons/pa1" {
		t.Fatalf("path = %q, want /openmeter/plans/plan1/addons/pa1", gotPath)
	}
}

func TestPlanAddons_Update(t *testing.T) {
	var gotBody UpsertPlanAddonRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/openmeter/plans/plan1/addons/pa1" {
			t.Errorf("path = %s", r.URL.Path)
		}

		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"id":"pa1","name":"Renamed","addon":{"id":"addon1"},"from_plan_phase":"trial","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	pa, err := c.PlanAddons.Update(t.Context(), "plan1", "pa1", UpsertPlanAddonRequest{
		Name:          "Renamed",
		FromPlanPhase: "trial",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if gotBody.Name != "Renamed" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if pa.Name != "Renamed" {
		t.Fatalf("unexpected plan-addon: %+v", pa)
	}
}

func TestPlanAddons_Delete(t *testing.T) {
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		gotPath = r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.PlanAddons.Delete(t.Context(), "plan1", "pa1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if gotPath != "/openmeter/plans/plan1/addons/pa1" {
		t.Fatalf("path = %q, want /openmeter/plans/plan1/addons/pa1", gotPath)
	}
}

func TestPlanAddons_EmptyIDGuards(t *testing.T) {
	// Every operation must reject an empty ID with ErrEmptyID before issuing any
	// request. Both the parent plan ID and the nested plan-addon ID are guarded.
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s: empty ID should be rejected client-side", r.URL.Path)
	})

	ctx := t.Context()

	if _, err := c.PlanAddons.List(ctx, "", PlanAddonListParams{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("List empty planID: err = %v, want ErrEmptyID", err)
	}
	if _, err := c.PlanAddons.Create(ctx, "", CreatePlanAddonRequest{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Create empty planID: err = %v, want ErrEmptyID", err)
	}
	if _, err := c.PlanAddons.Get(ctx, "", "pa1"); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Get empty planID: err = %v, want ErrEmptyID", err)
	}
	if _, err := c.PlanAddons.Get(ctx, "plan1", ""); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Get empty planAddonID: err = %v, want ErrEmptyID", err)
	}
	if _, err := c.PlanAddons.Update(ctx, "plan1", "", UpsertPlanAddonRequest{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Update empty planAddonID: err = %v, want ErrEmptyID", err)
	}
	if err := c.PlanAddons.Delete(ctx, "plan1", ""); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Delete empty planAddonID: err = %v, want ErrEmptyID", err)
	}
}

func TestPlanAddons_ListAll_Paginates(t *testing.T) {
	// The shared paginator must thread the parent plan ID through every page
	// request and walk both pages of the nested collection.
	pages := map[string]string{
		"1": `{"data":[{"id":"pa1","name":"n1","addon":{"id":"a1"},"from_plan_phase":"p","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"},{"id":"pa2","name":"n2","addon":{"id":"a2"},"from_plan_phase":"p","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"meta":{"page":{"number":1,"size":2,"total":3}}}`,
		"2": `{"data":[{"id":"pa3","name":"n3","addon":{"id":"a3"},"from_plan_phase":"p","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"meta":{"page":{"number":2,"size":2,"total":3}}}`,
	}
	var gotPaths []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, pages[r.URL.Query().Get("page[number]")])
	})

	var ids []string
	for pa, err := range c.PlanAddons.ListAll(t.Context(), "plan1", PlanAddonListParams{Page: &PageParams{Size: Int(2)}}) {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		ids = append(ids, pa.ID)
	}

	if got := strings.Join(ids, ","); got != "pa1,pa2,pa3" {
		t.Fatalf("ids = %q, want pa1,pa2,pa3", got)
	}
	for _, p := range gotPaths {
		if p != "/openmeter/plans/plan1/addons" {
			t.Fatalf("page request path = %q, want /openmeter/plans/plan1/addons", p)
		}
	}
}
