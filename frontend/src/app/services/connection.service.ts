import { Injectable, signal, computed, inject } from '@angular/core';
import { DatabaseConnection } from '../models/database.model';
import { QueryService } from './query.service';
import { firstValueFrom } from 'rxjs';

@Injectable({
    providedIn: 'root'
})
export class ConnectionService {
    private queryService = inject(QueryService);

    private connectionsSignal = signal<DatabaseConnection[]>([]);
    private activeConnectionSignal = signal<DatabaseConnection | null>(null);
    private sessionIdSignal = signal<string | null>(sessionStorage.getItem('db_session_id'));
    private isLoadingMetadataSignal = signal<boolean>(false);

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

        if (connection) {
            this.isLoadingMetadataSignal.set(true);
            try {
                const response = await firstValueFrom(this.queryService.initializeConnection(connection));
                if (response?.session_id) {
                    this.sessionIdSignal.set(response.session_id);
                    sessionStorage.setItem('db_session_id', response.session_id);
                }
            } catch (error) {
                console.error('Failed to initialize connection session', error);
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

    getConnection(id: string): DatabaseConnection | undefined {
        return this.connectionsSignal().find(c => c.id === id);
    }
}