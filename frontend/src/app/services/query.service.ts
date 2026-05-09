import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { DatabaseConnection, ApiResponse } from '../models/database.model';

@Injectable({
    providedIn: 'root'
})
export class QueryService {
    private connectionBaseUrl = '/api/connection';
    private queryBaseUrl = '/api/query';

    constructor(private http: HttpClient) { }

    testConnection(connection: DatabaseConnection): Observable<ApiResponse> {
        return this.http.post<ApiResponse>(`${this.connectionBaseUrl}/test`, connection);
    }

    initializeConnection(connection: DatabaseConnection): Observable<ApiResponse> {
        return this.http.post<ApiResponse>(`${this.connectionBaseUrl}/initialize`, connection);
    }

    executeQuery(sessionId: string, queryId: string, query: string): Observable<ApiResponse> {
        return this.http.post<ApiResponse>(`${this.queryBaseUrl}/execute`, {
            sessionId,
            queryId,
            query
        });
    }
}