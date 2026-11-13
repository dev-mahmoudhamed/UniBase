import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

export interface RedisKeyValueResponse {
    key: string;
    value: string;
    type: string;
    ttl: number;
    database: string;
}

export interface RedisKeyUpdateRequest {
    sessionId: string;
    database: string;
    oldKey: string;
    newKey: string;
    value: string;
    /** null keeps the current expiry, -1 removes it, >0 sets it in seconds */
    ttl: number | null;
}

@Injectable({
    providedIn: 'root'
})
export class RedisService {
    private baseUrl = '/api/redis';

    constructor(private http: HttpClient) { }

    getKeyValue(sessionId: string, database: string, key: string): Observable<RedisKeyValueResponse> {
        return this.http.post<{ data: RedisKeyValueResponse }>(`${this.baseUrl}/key/get`, {
            sessionId,
            database,
            key
        }).pipe(map(response => response.data));
    }

    updateKeyValue(request: RedisKeyUpdateRequest): Observable<any> {
        return this.http.post(`${this.baseUrl}/key/update`, request);
    }

    deleteKey(sessionId: string, database: string, key: string): Observable<any> {
        return this.http.post(`${this.baseUrl}/key/delete`, {
            sessionId,
            database,
            key
        });
    }
}