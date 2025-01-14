package billingworkersubscription

import "github.com/openmeterio/openmeter/openmeter/billing"

type patchOperation string

const (
	patchOpDelete patchOperation = "delete"
	patchOpUpdate patchOperation = "update"
)

type linePatch struct {
	Op          patchOperation
	InvoiceID   string
	LineID      string
	TargetState *billing.Line
}

func patchFromLine(op patchOperation, line *billing.Line) linePatch {
	if op == patchOpDelete {
		return linePatch{
			Op:        op,
			InvoiceID: line.InvoiceID,
			LineID:    line.ID,
		}
	}

	return linePatch{
		Op:          op,
		InvoiceID:   line.InvoiceID,
		LineID:      line.ID,
		TargetState: line,
	}
}
