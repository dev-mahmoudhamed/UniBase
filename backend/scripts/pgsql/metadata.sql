WITH schema_list AS (
    SELECT oid, nspname 
    FROM pg_namespace 
    WHERE nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast') 
      AND nspname NOT LIKE 'pg_temp_%'
)
SELECT json_build_object(
    'schemas', COALESCE(json_agg(
        json_build_object(
            'schema_name', s.nspname,
            'schema_contents', json_build_object(
                
                -- 1. TABLES (With Columns, Constraints, Indexes)
                'tables', COALESCE((
                    SELECT json_agg(
                        json_build_object(
                            'table_name', t.relname,
                            'columns', (
                                SELECT COALESCE(json_agg(
                                    json_build_object(
                                        'name', a.attname,
                                        'type', format_type(a.atttypid, a.atttypmod),
                                        'nullable', NOT a.attnotnull,
                                        'default', pg_get_expr(ad.adbin, ad.adrelid)
                                    ) ORDER BY a.attnum
                                ), '[]'::json)
                                FROM pg_attribute a
                                LEFT JOIN pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
                                WHERE a.attrelid = t.oid AND a.attnum > 0 AND NOT a.attisdropped
                            ),
                            'constraints', (
                                SELECT COALESCE(json_agg(
                                    json_build_object(
                                        'name', c.conname,
                                        'type', CASE c.contype 
                                            WHEN 'p' THEN 'PRIMARY KEY'
                                            WHEN 'f' THEN 'FOREIGN KEY'
                                            WHEN 'u' THEN 'UNIQUE'
                                            WHEN 'c' THEN 'CHECK'
                                            ELSE c.contype::text 
                                        END,
                                        'definition', pg_get_constraintdef(c.oid)
                                    )
                                ), '[]'::json)
                                FROM pg_constraint c
                                WHERE c.conrelid = t.oid
                            ),
                            'indexes', (
                                SELECT COALESCE(json_agg(
                                    json_build_object(
                                        'name', i.relname,
                                        'definition', pg_get_indexdef(idx.indexrelid)
                                    )
                                ), '[]'::json)
                                FROM pg_index idx
                                JOIN pg_class i ON i.oid = idx.indexrelid
                                WHERE idx.indrelid = t.oid
                            )
                        ) ORDER BY t.relname
                    )
                    FROM pg_class t
                    WHERE t.relnamespace = s.oid AND t.relkind = 'r'
                ), '[]'::json),

                -- 2. VIEWS (With Columns and Definition)
                'views', COALESCE((
                    SELECT json_agg(
                        json_build_object(
                            'view_name', v.relname,
                            'definition', pg_get_viewdef(v.oid, true),
                            'columns', (
                                SELECT COALESCE(json_agg(
                                    json_build_object(
                                        'name', a.attname,
                                        'type', format_type(a.atttypid, a.atttypmod)
                                    ) ORDER BY a.attnum
                                ), '[]'::json)
                                FROM pg_attribute a
                                WHERE a.attrelid = v.oid AND a.attnum > 0 AND NOT a.attisdropped
                            )
                        ) ORDER BY v.relname
                    )
                    FROM pg_class v
                    WHERE v.relnamespace = s.oid AND v.relkind = 'v'
                ), '[]'::json),

                -- 3. FUNCTIONS (With Arguments and Return Type)
                'functions', COALESCE((
                    SELECT json_agg(
                        json_build_object(
                            'function_name', p.proname,
                            'arguments', pg_get_function_arguments(p.oid),
                            'return_type', format_type(p.prorettype, null),
                            'language', l.lanname,
                            'volatility', CASE p.provolatile 
                                WHEN 'i' THEN 'IMMUTABLE' 
                                WHEN 's' THEN 'STABLE' 
                                WHEN 'v' THEN 'VOLATILE' 
                            END
                        ) ORDER BY p.proname
                    )
                    FROM pg_proc p
                    JOIN pg_language l ON p.prolang = l.oid
                    WHERE p.pronamespace = s.oid AND p.prokind = 'f'
                ), '[]'::json),

                -- 4. PROCEDURES
                'procedures', COALESCE((
                    SELECT json_agg(
                        json_build_object(
                            'procedure_name', p.proname,
                            'arguments', pg_get_function_arguments(p.oid)
                        ) ORDER BY p.proname
                    )
                    FROM pg_proc p
                    WHERE p.pronamespace = s.oid AND p.prokind = 'p'
                ), '[]'::json)
            )
        ) ORDER BY s.nspname
    ), '[]'::json)
) AS full_database_tree
FROM schema_list s;