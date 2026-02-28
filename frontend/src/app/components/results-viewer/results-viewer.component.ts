import { Component, input, signal, ChangeDetectionStrategy } from '@angular/core';
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

    toggleCompact(): void {
        this.isCompact.set(!this.isCompact());
    }

    isQueryResult(result: any): result is QueryResult {
        return result && 'columns' in result;
    }

    isQueryError(result: any): result is QueryError {
        return result && 'message' in result;
    }

    rowTrackBy(index: number, item: any): number {
        return index;
    }
}