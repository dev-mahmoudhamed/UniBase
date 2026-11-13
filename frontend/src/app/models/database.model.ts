export enum DatabaseProvider {
    POSTGRESQL = 'postgresql',
    MSSQL = 'mssql',
    MYSQL = 'mysql',
    MONGODB = 'mongodb',
    REDIS = 'redis'
}

export interface DatabaseConnection {
    id: string;
    name: string;
    provider: DatabaseProvider;
    host: string;
    port: number;
    database?: string;
    user?: string;
    password?: string;
    /** Shown as a tag; 'prod' adds a red stripe and asks before running write statements. */
    environment?: Environment;
    isActive?: boolean;
}

export type Environment = 'local' | 'dev' | 'staging' | 'prod';

export const ENVIRONMENTS: { value: Environment; label: string }[] = [
    { value: 'local', label: 'Local' },
    { value: 'dev', label: 'Dev' },
    { value: 'staging', label: 'Staging' },
    { value: 'prod', label: 'Prod' },
];

export function envLabel(connection: { environment?: Environment } | null | undefined): string {
    return ENVIRONMENTS.find(e => e.value === (connection?.environment ?? 'local'))!.label;
}

/** One table of results. Rows are positional, matching `columns` by index. */
export interface ResultSet {
    columns: string[];
    rows: unknown[][];
    rowCount: number;
    /** True when the server stopped reading at its row limit. */
    truncated: boolean;
}

export interface QueryResult {
    resultSets: ResultSet[];
    executionTime: number;
}

export interface QueryError {
    message: string;
    code?: string;
}

export interface ExecuteQueryResponse {
    status?: string;
    queryId?: string;
    results?: QueryResult;
    error?: string;
    message?: string;
}

export interface ConnectionResponse {
    message?: string;
    status: string;
    session_id?: string;
    metadata?: any;
}

export interface ProviderMetadata {
    id: string;
    name: string;
    displayName: string;
    icon: string;
    description: string;
    defaultPort: number;
    fields: ConnectionField[];
    features: ProviderFeatures;
}

export interface ConnectionField {
    key: string;
    label: string;
    type: 'text' | 'number' | 'password' | 'select';
    placeholder?: string;
    defaultValue?: any;
    required: boolean;
    options?: string[];
    helpText?: string;
}

export interface ProviderFeatures {
    supportsSSL: boolean;
    supportsMultipleDB: boolean;
    supportsSchemas: boolean;
    supportsProcedures: boolean;
    supportsFunctions: boolean;
    supportsViews: boolean;
}

export interface ProvidersResponse {
    providers: ProviderMetadata[];
    version: string;
}

export interface ApiResponse<T = any> {
    status: 'success' | 'error' | 'accepted';
    message?: string;
    data?: T;
    error?: string;
}
