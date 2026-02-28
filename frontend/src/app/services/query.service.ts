import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { DatabaseConnection, QueryResult, QueryError, ConnectionResponse, ExecuteQueryResponse } from '../models/database.model';

@Injectable({
    providedIn: 'root'
})
export class QueryService {
    private connectionBaseUrl = '/connection';
    private queryBaseUrl = '/query';

    constructor(private http: HttpClient) { }

    testConnection(connection: DatabaseConnection): Observable<QueryResult | QueryError> {
        return this.http.post<QueryResult | QueryError>(`${this.connectionBaseUrl}/test`, connection);
    }

    initializeConnection(connection: DatabaseConnection): Observable<ConnectionResponse> {
        return this.http.post<ConnectionResponse>(`${this.connectionBaseUrl}/initialize`, connection);
    }

    executeQuery(sessionId: string, queryId: string, query: string): Observable<ExecuteQueryResponse> {
        return this.http.post<ExecuteQueryResponse>(`${this.queryBaseUrl}/execute`, {
            sessionId,
            queryId,
            query
        });
    }

}