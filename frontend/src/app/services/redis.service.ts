import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

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
    ttl: number;
}

@Injectable({
    providedIn: 'root'
})
export class RedisService {
    private baseUrl = '/redis';

    constructor(private http: HttpClient) { }

    getKeyValue(sessionId: string, database: string, key: string): Observable<RedisKeyValueResponse> {
        return this.http.post<RedisKeyValueResponse>(`${this.baseUrl}/key/get`, {
            sessionId,
            database,
            key
        });
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
