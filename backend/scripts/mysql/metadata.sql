SELECT JSON_OBJECT(
    'databases', COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'database_name', s.SCHEMA_NAME,
                'database_contents', JSON_OBJECT(
                    'tables', COALESCE(
                        (SELECT JSON_ARRAYAGG(
                            JSON_OBJECT(
                                'table_name', t.TABLE_NAME,
                                'columns', COALESCE(
                                    (SELECT JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'name', c.COLUMN_NAME,
                                            'type', c.COLUMN_TYPE,
                                            'nullable', IF(c.IS_NULLABLE = 'YES', TRUE, FALSE),
                                            'default', c.COLUMN_DEFAULT
                                        )
                                    )
                                    FROM information_schema.columns c
                                    WHERE c.TABLE_SCHEMA = s.SCHEMA_NAME AND c.TABLE_NAME = t.TABLE_NAME),
                                    JSON_ARRAY()
                                ),
                                'indexes', COALESCE(
                                    (SELECT JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'name', stat.INDEX_NAME,
                                            'type', stat.INDEX_TYPE
                                        )
                                    )
                                    FROM information_schema.statistics stat
                                    WHERE stat.TABLE_SCHEMA = s.SCHEMA_NAME AND stat.TABLE_NAME = t.TABLE_NAME AND stat.SEQ_IN_INDEX = 1),
                                    JSON_ARRAY()
                                )
                            )
                        )
                        FROM information_schema.tables t
                        WHERE t.TABLE_SCHEMA = s.SCHEMA_NAME AND t.TABLE_TYPE = 'BASE TABLE'),
                        JSON_ARRAY()
                    ),
                    'views', COALESCE(
                        (SELECT JSON_ARRAYAGG(
                            JSON_OBJECT(
                                'view_name', v.TABLE_NAME,
                                'definition', v.VIEW_DEFINITION
                            )
                        )
                        FROM information_schema.views v
                        WHERE v.TABLE_SCHEMA = s.SCHEMA_NAME),
                        JSON_ARRAY()
                    ),
                    'functions', COALESCE(
                        (SELECT JSON_ARRAYAGG(
                            JSON_OBJECT(
                                'function_name', r.ROUTINE_NAME,
                                'return_type', r.DTD_IDENTIFIER
                            )
                        )
                        FROM information_schema.routines r
                        WHERE r.ROUTINE_SCHEMA = s.SCHEMA_NAME AND r.ROUTINE_TYPE = 'FUNCTION'),
                        JSON_ARRAY()
                    ),
                    'procedures', COALESCE(
                        (SELECT JSON_ARRAYAGG(
                            JSON_OBJECT(
                                'procedure_name', r.ROUTINE_NAME
                            )
                        )
                        FROM information_schema.routines r
                        WHERE r.ROUTINE_SCHEMA = s.SCHEMA_NAME AND r.ROUTINE_TYPE = 'PROCEDURE'),
                        JSON_ARRAY()
                    )
                )
            )
        )
        FROM information_schema.schemata s
        WHERE s.SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
        ),
        JSON_ARRAY()
    )
) AS full_database_tree;
