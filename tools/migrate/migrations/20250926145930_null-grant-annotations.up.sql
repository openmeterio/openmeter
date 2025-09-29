-- Now let's null annotations. This should be run together with the previous migration. They are in seperate files so there's no change migration runner would mistakenly run this without the previous migration succeeding.
UPDATE "grants" SET "annotations" = NULL;
