-- modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "subject_id" character(26) NULL;

-- Now let's find the matching subject_id for each entitlement by filtering for subject_key and namespace
-- We use DISTINCT ON to ensure we only get one row per entitlement (even though we have a unique constraint on key+namespace)
WITH tt AS (
    SELECT DISTINCT ON (e.id)
        e.id as entitlement_id,
        e.namespace,
        s.id as subject_id
    FROM entitlements e
    JOIN subjects s ON e.subject_key = s.key AND e.namespace = s.namespace
    ORDER BY e.id, s.created_at ASC
)
UPDATE entitlements e
SET subject_id = t.subject_id
FROM tt t
WHERE e.id = t.entitlement_id;

-- Now, let's create a new subject for each entitlement that doesn't have a subject_id and set the subject_id in the entitlements table
WITH new_subjects AS (
    INSERT INTO subjects (id, key, namespace, created_at)
    SELECT DISTINCT ON (e.subject_key, e.namespace)
        om_func_generate_ulid(),
        e.subject_key,
        e.namespace,
        e.created_at
    FROM entitlements e
    WHERE e.subject_id IS NULL
    RETURNING id, key, namespace
)
UPDATE entitlements e
SET subject_id = s.id
FROM new_subjects s
WHERE e.subject_key = s.key AND e.namespace = s.namespace AND e.subject_id IS NULL;

-- atlas:nolint MF104
ALTER TABLE "entitlements" ALTER COLUMN "subject_id" SET NOT NULL;

-- Now let's add the foreign key constraint
ALTER TABLE "entitlements" ADD CONSTRAINT "entitlements_subjects_entitlements" FOREIGN KEY ("subject_id") REFERENCES "subjects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

