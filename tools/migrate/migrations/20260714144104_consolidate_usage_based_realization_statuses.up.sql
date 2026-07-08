-- consolidate legacy usage-based realization statuses into the canonical realization branch
UPDATE "charge_usage_based"
SET "status_detailed" = CASE "status_detailed"
    WHEN 'active.partial_invoice.started' THEN 'active.realization.started'
    WHEN 'active.final_realization.started' THEN 'active.realization.started'
    WHEN 'active.partial_invoice.waiting_for_collection' THEN 'active.realization.waiting_for_collection'
    WHEN 'active.final_realization.waiting_for_collection' THEN 'active.realization.waiting_for_collection'
    WHEN 'active.partial_invoice.processing' THEN 'active.realization.processing'
    WHEN 'active.final_realization.processing' THEN 'active.realization.processing'
    WHEN 'active.partial_invoice.issuing' THEN 'active.realization.issuing'
    WHEN 'active.final_realization.issuing' THEN 'active.realization.issuing'
    WHEN 'active.partial_invoice.completed' THEN 'active.realization.completed'
    WHEN 'active.final_realization.completed' THEN 'active.realization.completed'
    ELSE "status_detailed"
END
WHERE "status_detailed" IN (
    'active.partial_invoice.started',
    'active.final_realization.started',
    'active.partial_invoice.waiting_for_collection',
    'active.final_realization.waiting_for_collection',
    'active.partial_invoice.processing',
    'active.final_realization.processing',
    'active.partial_invoice.issuing',
    'active.final_realization.issuing',
    'active.partial_invoice.completed',
    'active.final_realization.completed'
);
