import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { ApiResponse, ProviderMetadata, ProvidersResponse } from '../models/database.model';

@Injectable({
    providedIn: 'root'
})
export class ProviderService {
    private baseUrl = '/api/providers';

    private providersSignal = signal<ProviderMetadata[]>([]);
    private loadingSignal = signal<boolean>(false);
    private versionSignal = signal<string>('');

    providers = this.providersSignal.asReadonly();
    isLoading = this.loadingSignal.asReadonly();
    version = this.versionSignal.asReadonly();

    constructor(private http: HttpClient) { }

    loadProviders(): Observable<ApiResponse<ProvidersResponse>> {
        this.loadingSignal.set(true);
        return this.http.get<ApiResponse<ProvidersResponse>>(this.baseUrl).pipe(
            tap(response => {
                if (response.status === 'success' && response.data) {
                    this.providersSignal.set(response.data.providers || []);
                    this.versionSignal.set(response.data.version || '');
                } else {
                    this.providersSignal.set([]);
                }
                this.loadingSignal.set(false);
            })

        );
    }


    getProviderById(id: string): Observable<ProviderMetadata> {
        return this.http.get<ProviderMetadata>(`${this.baseUrl}/${id}`);
    }

    validateConfig(providerId: string, config: any): Observable<{ valid: boolean; errors?: string[] }> {
        return this.http.post<{ valid: boolean; errors?: string[] }>(
            `${this.baseUrl}/validate`,
            { providerId, config }
        );
    }

    getProvider(id: string): ProviderMetadata | undefined {
        const providers = this.providersSignal();
        if (!Array.isArray(providers)) return undefined;
        return providers.find(p => p.id === id);
    }


    getProviderIcon(id: string): string {
        const provider = this.getProvider(id);
        return provider?.icon || '';
    }

    getAvailableProviderIds(): string[] {
        return this.providersSignal().map(p => p.id);
    }

    isProviderSupported(id: string): boolean {
        return this.providersSignal().some(p => p.id === id);
    }
}