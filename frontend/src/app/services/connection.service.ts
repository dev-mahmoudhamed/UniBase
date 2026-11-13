import { Injectable, signal, computed, inject } from '@angular/core';
import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { DatabaseConnection } from '../models/database.model';
import { QueryService } from './query.service';
import { catchError, firstValueFrom, from, switchMap, throwError } from 'rxjs';

/**
 * Sessions live in server memory, so a server restart forgets them (401).
 * Reconnects the active connection and retries the request once. Queries are
 * not re-run automatically, because that could repeat an UPDATE or DELETE.
 */
export const sessionRetryInterceptor: HttpInterceptorFn = (req, next) => {
    const body = req.body as { sessionId?: string } | null;
    if (!body?.sessionId) return next(req);
    const connections = inject(ConnectionService);
    return next(req).pipe(catchError(err => {
        if (err?.status !== 401) return throwError(() => err);
        return from(connections.reconnect()).pipe(switchMap(sessionId => {
            if (!sessionId) return throwError(() => err);
            if (req.url.includes('/query/')) {
                return throwError(() => new HttpErrorResponse({
                    status: 401, error: { error: 'Reconnected to the server. Run the query again.' }
                }));
            }
            return next(req.clone({ body: { ...body, sessionId } }));
        }));
    }));
};

@Injectable({
    providedIn: 'root'
})
export class ConnectionService {
    private queryService = inject(QueryService);

    private connectionsSignal = signal<DatabaseConnection[]>([]);
    private activeConnectionSignal = signal<DatabaseConnection | null>(null);
    private sessionIdSignal = signal<string | null>(sessionStorage.getItem('db_session_id'));
    private isLoadingMetadataSignal = signal<boolean>(false);
    /** Why the last connect attempt failed, shown in the sidebar. */
    readonly connectionError = signal<string | null>(null);

    connections = computed(() => this.connectionsSignal());
    activeConnection = computed(() => this.activeConnectionSignal());
    sessionId = computed(() => this.sessionIdSignal());
    isLoadingMetadata = computed(() => this.isLoadingMetadataSignal());

    constructor() {
        this.loadConnections();
    }

    private loadConnections(): void {
        const stored = localStorage.getItem('db_connections');
        if (stored) {
            try {
                this.connectionsSignal.set(JSON.parse(stored));
            } catch (e) {
                console.error('Failed to load connections', e);
            }
        }
    }

    private saveConnections(): void {
        localStorage.setItem('db_connections', JSON.stringify(this.connectionsSignal()));
    }

    addConnection(connection: DatabaseConnection): void {
        this.connectionsSignal.update(connections => [...connections, connection]);
        this.saveConnections();
    }

    updateConnection(connection: DatabaseConnection): void {
        this.connectionsSignal.update(connections =>
            connections.map(c => c.id === connection.id ? connection : c)
        );
        this.saveConnections();
    }

    removeConnection(id: string): void {
        this.connectionsSignal.update(connections => connections.filter(c => c.id !== id));
        this.saveConnections();

        if (this.activeConnectionSignal()?.id === id) {
            this.setActiveConnection(null);
        }
    }

    async setActiveConnection(connection: DatabaseConnection | null): Promise<void> {
        this.activeConnectionSignal.set(connection);

        this.connectionError.set(null);
        if (connection) {
            this.isLoadingMetadataSignal.set(true);
            try {
                const response = await firstValueFrom(this.queryService.initializeConnection(connection)) as any;
                const data = response?.data;
                if (data?.session_id) {
                    this.sessionIdSignal.set(data.session_id);
                    sessionStorage.setItem('db_session_id', data.session_id);
                }

            } catch (error) {
                this.connectionError.set((error as any)?.error?.error || 'Could not connect to the server');
                this.sessionIdSignal.set(null);
                sessionStorage.removeItem('db_session_id');
            } finally {
                this.isLoadingMetadataSignal.set(false);
            }
        } else {
            this.sessionIdSignal.set(null);
            sessionStorage.removeItem('db_session_id');
        }
    }

    private reconnecting: Promise<string | null> | null = null;

    /** Opens a new server session for the active connection (e.g. after a server restart). */
    reconnect(): Promise<string | null> {
        const connection = this.activeConnectionSignal();
        if (!connection) return Promise.resolve(null);
        this.reconnecting ??= this.setActiveConnection(connection)
            .then(() => this.sessionIdSignal())
            .finally(() => this.reconnecting = null);
        return this.reconnecting;
    }

    getConnection(id: string): DatabaseConnection | undefined {
        return this.connectionsSignal().find(c => c.id === id);
    }
}