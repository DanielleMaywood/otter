-- :one
-- $1: oid
select
    typname as "name",
    typtype as "type",
    typnotnull as "not_null"
from pg_type where oid = $1 limit 1
