import { Component, inject, input, signal, computed, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RedisService, RedisKeyValueResponse } from '../../services/redis.service';
import { ConnectionService } from '../../services/connection.service';

@Component({
    selector: 'app-redis-viewer',
    standalone: true,
    imports: [FormsModule],
    templateUrl: './redis-viewer.component.html',
    styleUrls: ['./redis-viewer.component.scss']
})
export class RedisViewerComponent {
    selectedKey = input<{ database: string; key: string } | null>(null);

    private redisService = inject(RedisService);
    private connectionService = inject(ConnectionService);

    isLoading = signal(false);
    isSaving = signal(false);
    isDeleting = signal(false);
    errorMessage = signal<string | null>(null);
    successMessage = signal<string | null>(null);

    keyData = signal<RedisKeyValueResponse | null>(null);

    // Editable fields
    editKey = signal('');
    editValue = signal('');
    editTTL = signal(-1);

    // Track dirty state
    isDirty = computed(() => {
        const data = this.keyData();
        if (!data) return false;
        return this.editKey() !== data.key ||
            this.editValue() !== data.value ||
            this.editTTL() !== data.ttl;
    });

    constructor() {
        effect(() => {
            const selected = this.selectedKey();
            if (selected) {
                this.loadKeyValue(selected.database, selected.key);
            } else {
                this.keyData.set(null);
            }
        });
    }

    loadKeyValue(database: string, key: string): void {
        const sessionId = this.connectionService.sessionId();
        if (!sessionId) return;

        this.isLoading.set(true);
        this.errorMessage.set(null);
        this.successMessage.set(null);

        this.redisService.getKeyValue(sessionId, database, key).subscribe({
            next: (data) => {
                this.keyData.set(data);
                this.editKey.set(data.key);
                this.editValue.set(data.value);
                this.editTTL.set(data.ttl);
                this.isLoading.set(false);
            },
            error: (err) => {
                this.errorMessage.set(err?.error?.error || 'Failed to load key value');
                this.isLoading.set(false);
            }
        });
    }

    saveChanges(): void {
        const sessionId = this.connectionService.sessionId();
        const data = this.keyData();
        if (!sessionId || !data) return;

        this.isSaving.set(true);
        this.errorMessage.set(null);
        this.successMessage.set(null);

        this.redisService.updateKeyValue({
            sessionId,
            database: data.database,
            oldKey: data.key,
            newKey: this.editKey(),
            value: this.editValue(),
            ttl: this.editTTL()
        }).subscribe({
            next: () => {
                this.successMessage.set('Key updated successfully');
                // Update the local data to reflect saved state
                this.keyData.set({
                    ...data,
                    key: this.editKey(),
                    value: this.editValue(),
                    ttl: this.editTTL()
                });
                this.isSaving.set(false);
                // Auto-clear success message
                setTimeout(() => this.successMessage.set(null), 3000);
            },
            error: (err) => {
                this.errorMessage.set(err?.error?.error || 'Failed to save changes');
                this.isSaving.set(false);
            }
        });
    }

    deleteKey(): void {
        const sessionId = this.connectionService.sessionId();
        const data = this.keyData();
        if (!sessionId || !data) return;

        if (!confirm(`Are you sure you want to delete key "${data.key}"?`)) return;

        this.isDeleting.set(true);
        this.errorMessage.set(null);

        this.redisService.deleteKey(sessionId, data.database, data.key).subscribe({
            next: () => {
                this.keyData.set(null);
                this.isDeleting.set(false);
                this.successMessage.set('Key deleted successfully');
            },
            error: (err) => {
                this.errorMessage.set(err?.error?.error || 'Failed to delete key');
                this.isDeleting.set(false);
            }
        });
    }

    resetChanges(): void {
        const data = this.keyData();
        if (!data) return;
        this.editKey.set(data.key);
        this.editValue.set(data.value);
        this.editTTL.set(data.ttl);
        this.errorMessage.set(null);
        this.successMessage.set(null);
    }

    formatTTL(ttl: number): string {
        if (ttl === -1) return 'No expiry';
        if (ttl === -2) return 'Key does not exist';
        if (ttl < 60) return `${ttl}s`;
        if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
        const hours = Math.floor(ttl / 3600);
        const mins = Math.floor((ttl % 3600) / 60);
        return `${hours}h ${mins}m`;
    }

    getTypeColor(type: string): string {
        switch (type) {
            case 'string': return '#98c379';
            case 'hash': return '#e5c07b';
            case 'list': return '#61afef';
            case 'set': return '#c678dd';
            case 'zset': return '#56b6c2';
            default: return '#abb2bf';
        }
    }

    isJsonValue(): boolean {
        const value = this.editValue();
        try {
            JSON.parse(value);
            return true;
        } catch {
            return false;
        }
    }

    formatValue(): void {
        try {
            const parsed = JSON.parse(this.editValue());
            this.editValue.set(JSON.stringify(parsed, null, 2));
        } catch { }
    }
}
