-- IDs are accepted by the previous writer and readers. Keep resolved identities
-- on rollback: restoring keys could retarget a Rule after a feature key is reused.
SELECT 1;
