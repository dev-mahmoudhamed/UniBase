-- name: databases
SELECT 
    name,
    database_id,
    CAST(CASE WHEN database_id <= 4 THEN 1 ELSE 0 END AS BIT) AS is_system
FROM sys.databases
WHERE state = 0
FOR JSON PATH;

-- name: tables
SELECT t.object_id, t.name
FROM sys.tables t
WHERE t.is_ms_shipped = 0
FOR JSON PATH;

-- name: columns
SELECT c.object_id, c.name AS column_name, ty.name AS data_type
FROM sys.columns c
JOIN sys.types ty ON c.user_type_id = ty.user_type_id
FOR JSON PATH;

-- name: keys
SELECT kc.parent_object_id AS object_id, kc.name, kc.type_desc
FROM sys.key_constraints kc
FOR JSON PATH;

-- name: indexes
SELECT i.object_id, i.name, i.type_desc
FROM sys.indexes i
WHERE i.is_primary_key = 0
  AND i.is_unique_constraint = 0
  AND i.name IS NOT NULL
FOR JSON PATH;

-- name: views
SELECT object_id, name
FROM sys.views
WHERE is_ms_shipped = 0
FOR JSON PATH;

-- name: procedures
SELECT object_id, name
FROM sys.procedures
WHERE is_ms_shipped = 0
FOR JSON PATH;

-- name: triggers
SELECT object_id, name
FROM sys.triggers
WHERE parent_class = 0
FOR JSON PATH;