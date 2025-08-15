UPDATE billing_invoice_lines
    SET annotations = annotations || '{"billing.subscription.sync.force-continuous-lines": true}'
    WHERE
         (annotations is not null and annotations <> 'null'::jsonb) and (annotations -> 'billing.subscription.sync.ignore' = 'true'::jsonb);
