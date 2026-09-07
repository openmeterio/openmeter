package billingadapter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
)

type validationIssueWithDedupe struct {
	issue billing.ValidationIssue
	hash  []byte
}

func issueDedupeHash(issue billing.ValidationIssue) ([]byte, error) {
	algo := sha256.New()

	algo.Write([]byte(issue.Severity))
	algo.Write([]byte(issue.Code))
	algo.Write([]byte(issue.Message))
	algo.Write([]byte(issue.Component))
	algo.Write([]byte(issue.Path))

	if len(issue.Attributes) > 0 {
		attributes, err := json.Marshal(issue.Attributes)
		if err != nil {
			return nil, fmt.Errorf("marshaling attributes: %w", err)
		}

		algo.Write([]byte{0})
		algo.Write(attributes)
	}

	return algo.Sum(nil), nil
}

// persistValidationIssues persists the validation issues for the given invoice, it will remove any
// existing issues that are not present in the new list. It relies on consistent hashing to deduplicate
// issues.
func (a *adapter) persistValidationIssues(ctx context.Context, invoice billing.InvoiceID, issues []billing.ValidationIssue) error {
	hashedIssues, err := lo.MapErr(issues, func(issue billing.ValidationIssue, _ int) (validationIssueWithDedupe, error) {
		hash, err := issueDedupeHash(issue)
		if err != nil {
			return validationIssueWithDedupe{}, fmt.Errorf("hashing validation issue: %w", err)
		}

		return validationIssueWithDedupe{
			issue: issue,
			hash:  hash,
		}, nil
	})
	if err != nil {
		return err
	}

	hashedIssues = lo.FindUniquesBy(
		hashedIssues,
		func(issue validationIssueWithDedupe) string {
			return string(issue.hash)
		},
	)

	err = a.db.BillingInvoiceValidationIssue.Update().
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

		if len(issue.Attributes) > 0 {
			c.SetAttributes(issue.Attributes)
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

// IntropectValidationIssues returns the validation issues for the given invoice, this is not
// exposed via the adpter interface, as it's only used by tests to validate the state of the
// database.
func (a *adapter) IntrospectValidationIssues(ctx context.Context, invoice billing.InvoiceID) (billing.ValidationIssues, error) {
	issues, err := a.db.BillingInvoiceValidationIssue.Query().
		Where(billinginvoicevalidationissue.InvoiceID(invoice.ID)).
		Where(billinginvoicevalidationissue.Namespace(invoice.Namespace)).
		Order(db.Asc(billinginvoicevalidationissue.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return lo.Map(issues, func(issue *db.BillingInvoiceValidationIssue, _ int) billing.ValidationIssue {
		return billing.ValidationIssue{
			ID:        issue.ID,
			CreatedAt: issue.CreatedAt.In(time.UTC),
			UpdatedAt: issue.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(issue.DeletedAt, time.UTC),

			Severity:   issue.Severity,
			Message:    issue.Message,
			Code:       lo.FromPtr(issue.Code),
			Component:  billing.ComponentName(issue.Component),
			Path:       lo.FromPtr(issue.Path),
			Attributes: issue.Attributes,
		}
	}), nil
}
