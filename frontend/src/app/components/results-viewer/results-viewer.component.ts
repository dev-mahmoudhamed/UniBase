import { Component, computed, effect, input, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { QueryResult, QueryError } from '../../models/database.model';

@Component({
    selector: 'app-results-viewer',
    standalone: true,
    imports: [CommonModule, ScrollingModule],
    templateUrl: './results-viewer.component.html',
    styleUrls: ['./results-viewer.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ResultsViewerComponent {
    result = input<QueryResult | QueryError | null>(null);
    isLoading = input(false);
    isCompact = signal(false);
    activeSet = signal(0);

    queryResult = computed(() => {
        const r = this.result();
        return r && 'resultSets' in r ? r : null;
    });
    errorMessage = computed(() => {
        const r = this.result();
        return r && 'message' in r ? r.message : null;
    });
    currentSet = computed(() => this.queryResult()?.resultSets[this.activeSet()] ?? null);

    /** Column widths in px from the header and the first 100 values, so short IDs stay narrow. */
    widths = computed(() => {
        const set = this.currentSet();
        if (!set) return [];
        const sample = set.rows.slice(0, 100);
        return set.columns.map((column, i) => {
            let chars = column.length;
            for (const row of sample) chars = Math.max(chars, this.formatValue(row[i]).length);
            return Math.round(Math.min(360, Math.max(72, chars * 8.2 + 30)));
        });
    });

    constructor() {
        // A new result starts on its first result set.
        effect(() => {
            this.result();
            this.activeSet.set(0);
        });
    }

    toggleCompact(): void {
        this.isCompact.set(!this.isCompact());
    }

    formatValue(val: unknown): string {
        if (val === null || val === undefined) return 'NULL';
        if (typeof val === 'object') {
            try {
                return JSON.stringify(val);
            } catch {
                return '[Object]';
            }
        }
        return String(val);
    }
}
