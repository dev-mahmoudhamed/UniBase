import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { ProviderMetadata, ProvidersResponse } from '../models/database.model';

@Injectable({
    providedIn: 'root'
})
export class ProviderService {
    private baseUrl = '/api/providers';

    // Signals for reactive state
    private providersSignal = signal<ProviderMetadata[]>([]);
    private loadingSignal = signal<boolean>(false);
    private versionSignal = signal<string>('');

    // Public readonly signals
    providers = this.providersSignal.asReadonly();
    isLoading = this.loadingSignal.asReadonly();
    version = this.versionSignal.asReadonly();

    constructor(private http: HttpClient) { }

    loadProviders(): Observable<ProvidersResponse> {
        this.loadingSignal.set(true);
        return this.http.get<ProvidersResponse>(this.baseUrl).pipe(
            tap(response => {
                this.providersSignal.set(response.providers);
                this.versionSignal.set(response.version);
                this.loadingSignal.set(false);
            })
        );
    }

    // Get a specific provider by ID
    getProviderById(id: string): Observable<ProviderMetadata> {
        return this.http.get<ProviderMetadata>(`${this.baseUrl}/${id}`);
    }

    // Validate provider configuration
    validateConfig(providerId: string, config: any): Observable<{ valid: boolean; errors?: string[] }> {
        return this.http.post<{ valid: boolean; errors?: string[] }>(
            `${this.baseUrl}/validate`,
            { providerId, config }
        );
    }

    // Get provider from local cache (signal)
    getProvider(id: string): ProviderMetadata | undefined {
        return this.providersSignal().find(p => p.id === id);
    }

    // Get provider icon
    getProviderIcon(id: string): string {
        const provider = this.getProvider(id);
        return provider?.icon || '';
    }

    // Get available provider IDs
    getAvailableProviderIds(): string[] {
        return this.providersSignal().map(p => p.id);
    }

    // Check if provider is supported
    isProviderSupported(id: string): boolean {
        return this.providersSignal().some(p => p.id === id);
    }
}