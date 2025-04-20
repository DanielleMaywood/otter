-- :many
-- $1: schema
-- $2: relation
select is_nullable = 'YES' from information_schema.columns where table_schema = $1 and table_name = $2 order by ordinal_position
