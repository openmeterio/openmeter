package billing

const (
	// AnnotationSubscriptionSyncIgnore is used to mark a line or hierarchy as ignored in subscription syncing.
	// Should be used in case there is a breaking change in the subscription synchronization process, preventing billing
	// from issuing credit notes for the past periods.
	AnnotationSubscriptionSyncIgnore = "billing.subscription.sync.ignore"

	// AnnotationSubscriptionSyncForceContinuousLines is used to force the creation of continuous subscription item lines.
	// If the sync process finds a previously existing line with this annotation, and the next line generated will not start at the end of the previously
	// found line, the sync process will adjust the start of the next line to the end of the previously found line, so that we don't have gaps in the
	// invoices.
	AnnotationSubscriptionSyncForceContinuousLines = "billing.subscription.sync.force-continuous-lines"
)
