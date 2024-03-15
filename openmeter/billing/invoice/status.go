// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package invoice

import "github.com/samber/lo"

type InvoiceStatus string

const (
	// InvoiceStatusCreated is the status of an invoice that has been created.
	InvoiceStatusCreated InvoiceStatus = "created"
	// InvoiceStatusDraft is the status of an invoice that is in draft both on OpenMeter and the provider side.
	InvoiceStatusDraft InvoiceStatus = "draft"
	// InvoiceStatusDraftSync is the status of an invoice that is being synced with the provider.
	InvoiceStatusDraftSync InvoiceStatus = "draft_sync"
	// InvoiceStatusDraftSyncFailed is the status of an invoice that failed to sync with the provider.
	InvoiceStatusDraftSyncFailed InvoiceStatus = "draft_sync_failed"
	// InvoiceStatusIssuing is the status of an invoice that is being issued.
	InvoiceStatusIssuing InvoiceStatus = "issuing"
	// InvoiceStatusIssued is the status of an invoice that has been issued both on OpenMeter and provider side.
	InvoiceStatusIssued InvoiceStatus = "issued"
	// InvoiceStatusIssuingFailed is the status of an invoice that failed to issue on the provider or OpenMeter side.
	InvoiceStatusIssuingFailed InvoiceStatus = "issuing_failed"
	// InvoiceStatusManualApprovalNeeded is the status of an invoice that needs manual approval. (due to AutoApprove is disabled)
	InvoiceStatusManualApprovalNeeded InvoiceStatus = "manual_approval_needed"
)

func (s InvoiceStatus) Values() []string {
	return lo.Map(
		[]InvoiceStatus{
			InvoiceStatusCreated,
			InvoiceStatusDraft,
			InvoiceStatusDraftSync,
			InvoiceStatusDraftSyncFailed,
			InvoiceStatusIssuing,
			InvoiceStatusIssued,
			InvoiceStatusIssuingFailed,
			InvoiceStatusManualApprovalNeeded,
		},
		func(item InvoiceStatus, _ int) string {
			return string(item)
		},
	)
}
