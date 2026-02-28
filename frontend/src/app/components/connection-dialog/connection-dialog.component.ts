import { Component, Output, EventEmitter, OnInit, inject, signal, computed, input } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ConnectionService } from '../../services/connection.service';
import { ProviderService } from '../../services/provider.service';
import { DatabaseConnection, ProviderMetadata, ConnectionField } from '../../models/database.model';
import { QueryService } from '../../services/query.service';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-connection-dialog',
    standalone: true,
    imports: [FormsModule],
    templateUrl: './connection-dialog.component.html',
    styleUrls: ['./connection-dialog.component.scss']
})
export class ConnectionDialogComponent implements OnInit {
    private connectionService = inject(ConnectionService);
    private providerService = inject(ProviderService);
    private queryService = inject(QueryService);

    @Output() close = new EventEmitter<void>();

    // Input for editing an existing connection
    editConnection = input<DatabaseConnection | null>(null);
    isEditMode = computed(() => !!this.editConnection());

    // Signals
    selectedProviderId = signal<string>('');
    connection = signal<Partial<DatabaseConnection>>({});
    errors = signal<string[]>([]);
    testStatus = signal<{ loading: boolean; message: string | null; success: boolean | null }>({
        loading: false,
        message: null,
        success: null
    });

    // Computed values from provider service
    providers = this.providerService.providers;
    selectedProvider = computed(() => {
        const id = this.selectedProviderId();
        return this.providerService.getProvider(id);
    });

    fields = computed(() => {
        return this.selectedProvider()?.fields || [];
    });

    ngOnInit(): void {
        const editing = this.editConnection();
        if (editing) {
            // Edit mode: pre-populate with existing connection data
            this.selectedProviderId.set(editing.provider);
            this.connection.set({ ...editing });
        } else {
            // New connection mode
            const availableProviders = this.providers();
            if (availableProviders.length > 0) {
                this.selectedProviderId.set(availableProviders[0].id);
                this.onProviderChange();
            }
        }
    }

    onProviderChange(): void {
        const provider = this.selectedProvider();
        if (!provider) return;

        const initialConnection: Partial<DatabaseConnection> = {
            provider: provider.id as any
        };

        // Set default values from provider metadata
        provider.fields.forEach(field => {
            if (field.defaultValue !== undefined) {
                (initialConnection as any)[field.key] = field.defaultValue;
            }
        });

        this.connection.set(initialConnection);
        this.testStatus.set({ loading: false, message: null, success: null });
        this.errors.set([]);
    }

    onTest(): void {
        this.errors.set([]);
        this.testStatus.set({ loading: true, message: 'Testing connection...', success: null });

        const validationErrors = this.validateConnection();
        if (validationErrors.length > 0) {
            this.errors.set(validationErrors);
            this.testStatus.set({ loading: false, message: 'Validation failed', success: false });
            return;
        }

        const testConn: DatabaseConnection = {
            ...this.connection() as DatabaseConnection,
            id: 'test_temp',
            provider: this.selectedProviderId() as any
        };

        this.queryService.testConnection(testConn)
            .pipe(finalize(() => this.testStatus.update(s => ({ ...s, loading: false }))))
            .subscribe({
                next: (result) => {
                    if (result && 'message' in result && !('columns' in result)) {
                        this.testStatus.set({ loading: false, message: result.message, success: true });
                    } else if (result) {
                        this.testStatus.set({ loading: false, message: 'Connection successful', success: true });
                    } else {
                        this.testStatus.set({ loading: false, message: 'No response from server', success: false });
                    }
                },
                error: (err) => {
                    this.testStatus.set({
                        loading: false,
                        message: err?.error?.error || err?.message || 'Connection failed',
                        success: false
                    });
                }
            });
    }

    onSave(): void {
        this.errors.set([]);
        this.testStatus.set({ loading: true, message: 'Initializing connection...', success: null });

        const validationErrors = this.validateConnection();
        if (validationErrors.length > 0) {
            this.errors.set(validationErrors);
            this.testStatus.set({ loading: false, message: 'Validation failed', success: false });
            return;
        }

        const connectionData: DatabaseConnection = {
            ...this.connection() as DatabaseConnection,
            id: 'init_temp',
            provider: this.selectedProviderId() as any
        };

        this.queryService.initializeConnection(connectionData)
            .pipe(finalize(() => this.testStatus.update(s => ({ ...s, loading: false }))))
            .subscribe({
                next: (response) => {
                    if (!response) {
                        this.testStatus.set({ loading: false, message: 'No response from server', success: false });
                        return;
                    }
                    const editing = this.editConnection();
                    if (editing) {
                        // Update mode
                        const updatedConnection: DatabaseConnection = {
                            ...connectionData,
                            id: editing.id,
                        };
                        this.connectionService.updateConnection(updatedConnection);
                    } else {
                        // New connection mode
                        const newConnection: DatabaseConnection = {
                            ...connectionData,
                            id: response.session_id || this.generateId()
                        };
                        this.connectionService.addConnection(newConnection);
                    }
                    this.close.emit();
                },
                error: (err) => {
                    this.testStatus.set({
                        loading: false,
                        message: err?.error?.error || err?.message || 'Initialization failed',
                        success: false
                    });
                }
            });
    }

    onCancel(): void {
        this.close.emit();
    }

    private validateConnection(): string[] {
        const errors: string[] = [];
        const provider = this.selectedProvider();
        if (!provider) {
            return ['No provider selected'];
        }

        const conn = this.connection();
        provider.fields.forEach(field => {
            if (!field.required) return;

            const value = (conn as any)[field.key];
            if (!value && value !== 0) {
                errors.push(`${field.label} is required`);
            }
        });

        return errors;
    }

    private generateId(): string {
        return `conn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    // Helper for template
    getFieldValue(key: string): any {
        return (this.connection() as any)[key];
    }

    setFieldValue(key: string, value: any): void {
        this.connection.update(conn => ({
            ...conn,
            [key]: value
        }));
    }
}