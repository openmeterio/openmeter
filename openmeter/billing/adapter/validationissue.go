package billingadapter

import (
	"context"
	"crypto/sha256"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type validationIssueWithDedupe struct {
	issue billing.ValidationIssue
	hash  []byte
}

func issueDedupeHash(issue billing.ValidationIssue) []byte {
	algo := sha256.New()

	algo.Write([]byte(issue.Severity))
	algo.Write([]byte(issue.Code))
	algo.Write([]byte(issue.Message))
	algo.Write([]byte(issue.Component))
	algo.Write([]byte(issue.Path))
	return algo.Sum(nil)
}

// persistValidationIssues persists the validation issues for the given invoice, it will remove any
// existing issues that are not present in the new list. It relies on consistent hashing to deduplicate
// issues.
func (a *adapter) persistValidationIssues(ctx context.Context, invoice billing.InvoiceID, issues []billing.ValidationIssue) error {
	// FIXME (pmarton): Why do we need to deduplicate issues?
	hashedIssues := dedupeIssues(
		lo.Map(issues, func(issue billing.ValidationIssue, _ int) validationIssueWithDedupe {
			return validationIssueWithDedupe{
				issue: issue,
				hash:  issueDedupeHash(issue),
			}
		}),
	)

	err := a.db.BillingInvoiceValidationIssue.Update().
		Where(billinginvoicevalidationissue.InvoiceID(invoice.ID)).
		Where(billinginvoicevalidationissue.Namespace(invoice.Namespace)).
		Where(billinginvoicevalidationissue.DedupeHashNotIn(
			lo.Map(hashedIssues, func(hashedIssue validationIssueWithDedupe, _ int) []byte {
				return hashedIssue.hash
			})...)).
		Where(billinginvoicevalidationissue.DeletedAtIsNil()).
		SetDeletedAt(clock.Now()).
		Exec(ctx)
	if err != nil {
		return err
	}

	return a.db.BillingInvoiceValidationIssue.MapCreateBulk(hashedIssues, func(c *db.BillingInvoiceValidationIssueCreate, i int) {
		hash := hashedIssues[i].hash
		issue := hashedIssues[i].issue

		c.SetNamespace(invoice.Namespace).
			SetInvoiceID(invoice.ID).
			SetSeverity(issue.Severity).
			SetMessage(issue.Message).
			SetComponent(string(issue.Component)).
			SetDedupeHash(hash)
		if issue.Code != "" {
			c.SetCode(issue.Code)
		}

		if issue.Path != "" {
			c.SetPath(issue.Path)
		}
	}).OnConflict(
		sql.ConflictColumns(
			billinginvoicevalidationissue.FieldNamespace,
			billinginvoicevalidationissue.FieldInvoiceID,
			billinginvoicevalidationissue.FieldDedupeHash,
		),
	).
		UpdateNewValues().
		Update(func(u *db.BillingInvoiceValidationIssueUpsert) {
			u.ClearDeletedAt()
			u.SetUpdatedAt(clock.Now())
		}).Exec(ctx)
}

type ValidationIssueWithDBMeta struct {
	billing.ValidationIssue

	ID        string     `json:"id"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// IntropectValidationIssues returns the validation issues for the given invoice, this is not
// exposed via the adpter interface, as it's only used by tests to validate the state of the
// database.
func (a *adapter) IntrospectValidationIssues(ctx context.Context, invoice billing.InvoiceID) ([]ValidationIssueWithDBMeta, error) {
	issues, err := a.db.BillingInvoiceValidationIssue.Query().
		Where(billinginvoicevalidationissue.InvoiceID(invoice.ID)).
		Where(billinginvoicevalidationissue.Namespace(invoice.Namespace)).
		Order(db.Asc(billinginvoicevalidationissue.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return lo.Map(issues, func(issue *db.BillingInvoiceValidationIssue, _ int) ValidationIssueWithDBMeta {
		return ValidationIssueWithDBMeta{
			ValidationIssue: billing.ValidationIssue{
				Severity:  issue.Severity,
				Message:   issue.Message,
				Code:      lo.FromPtrOr(issue.Code, ""),
				Component: billing.ComponentName(issue.Component),
				Path:      lo.FromPtrOr(issue.Path, ""),
			},
			ID:        issue.ID,
			DeletedAt: issue.DeletedAt,
		}
	}), nil
}

// dedupeIssues deduplicates the given list of issues by hash.
func dedupeIssues(issues []validationIssueWithDedupe) []validationIssueWithDedupe {
	issueSet := map[string]struct{}{}

	for _, hashedIssue := range issues {
		issueSet[string(hashedIssue.hash)] = struct{}{}
	}

	var dedupedIssues []validationIssueWithDedupe

	for _, hashedIssue := range issues {
		if _, ok := issueSet[string(hashedIssue.hash)]; ok {
			dedupedIssues = append(dedupedIssues, hashedIssue)
		}
	}

	return dedupedIssues
}
