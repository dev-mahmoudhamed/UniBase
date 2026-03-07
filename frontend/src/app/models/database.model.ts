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
    isActive?: boolean;
}

export interface QueryResult {
    columns: string[];
    rows: any[];
    rowCount: number;
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