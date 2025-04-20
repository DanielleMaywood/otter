-- :one
-- $1: schema
-- $2: relation
-- $3: column_name
select is_nullable = 'YES' from information_schema.columns where table_schema = $1 and table_name = $2 and column_name = $3 limit 1
