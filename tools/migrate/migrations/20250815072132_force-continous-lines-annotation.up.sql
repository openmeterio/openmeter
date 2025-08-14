UPDATE billing_invoice_lines
    SET annotations = annotations || '{"billing.subscription.sync.force-continous-lines": true}'
    WHERE
         (annotations is not null or annotations <> 'null'::jsonb) and (annotations -> 'billing.subscription.sync.ignore' = 'true'::jsonb);
