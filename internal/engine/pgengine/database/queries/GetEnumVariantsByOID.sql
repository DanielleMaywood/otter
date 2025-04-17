-- :many
-- $1: oid
select enumlabel from pg_enum where enumtypid = $1 order by enumsortorder
