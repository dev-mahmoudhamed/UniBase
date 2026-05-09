import { Component, inject, input, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ExplorerService, MongoCollectionData } from '../../services/explorer.service';
import { ConnectionService } from '../../services/connection.service';
import { MongoQueryBarComponent, MongoQueryOptions } from './mongo-query-bar/mongo-query-bar.component';
import { MongoDocumentCardComponent } from './mongo-document-card/mongo-document-card.component';

@Component({
  selector: 'app-mongo-viewer',
  standalone: true,
  imports: [CommonModule, FormsModule, MongoQueryBarComponent, MongoDocumentCardComponent],
  templateUrl: './mongo-viewer.component.html',
  styleUrls: ['./mongo-viewer.component.scss']
})
export class MongoViewerComponent {
  collectionData = input<MongoCollectionData | null>(null);

  private explorerService = inject(ExplorerService);
  private connectionService = inject(ConnectionService);

  // Local state
  localDocuments = signal<Record<string, any>[]>([]);
  isLoading = signal(false);
  errorMessage = signal<string | null>(null);
  successMessage = signal<string | null>(null);

  // Pagination state
  currentPage = signal(1);
  currentCount = signal(0);
  lastQueryOptions = signal<MongoQueryOptions>({
    filter: '{}',
    projection: '',
    sort: '',
    limit: 20
  });

  columns = signal<string[]>([]);

  constructor() {
    // Sync initial data from input
    effect(() => {
      const data = this.collectionData();
      if (data) {
        this.localDocuments.set(data.documents ? [...data.documents] : []);
        this.currentCount.set(data.count ?? 0);
        this.columns.set(data.columns || []);
        this.currentPage.set(1);
        this.errorMessage.set(null);
        this.successMessage.set(null);
      }
    }, { allowSignalWrites: true });
  }

  onQueryExecuted(options: MongoQueryOptions): void {
    this.lastQueryOptions.set(options);
    this.currentPage.set(1);
    this.fetchData();
  }

  fetchData(): void {
    const data = this.collectionData();
    const sessionId = this.connectionService.sessionId();
    const options = this.lastQueryOptions();

    if (!data || !sessionId) return;

    this.isLoading.set(true);
    this.errorMessage.set(null);

    const skip = (this.currentPage() - 1) * options.limit;

    this.explorerService.getCollectionData(
      sessionId, 
      data.collection, 
      { database: data.database }, 
      {
        filter: options.filter,
        projection: options.projection,
        sort: options.sort,
        skip: skip,
        limit: options.limit
      }
    ).subscribe({
      next: (resp: any) => {
        this.localDocuments.set(resp.documents || []);
        this.currentCount.set(resp.count ?? 0);
        this.columns.set(resp.columns || []);
        this.isLoading.set(false);
        // Scroll to top of the viewer container if possible
        const el = document.querySelector('.documents-scroll-container');
        if (el) el.scrollTo({ top: 0, behavior: 'smooth' });
      },
      error: (err: any) => {
        this.errorMessage.set(err?.error?.error || 'Query failed');
        this.isLoading.set(false);
      }
    });
  }

  nextPage(): void {
    this.currentPage.set(this.currentPage() + 1);
    this.fetchData();
  }

  prevPage(): void {
    if (this.currentPage() > 1) {
      this.currentPage.set(this.currentPage() - 1);
      this.fetchData();
    }
  }

  getPagingRange(): string {
    const total = this.currentCount();
    if (total === 0) return '0 – 0';
    const limit = this.lastQueryOptions().limit;
    const start = (this.currentPage() - 1) * limit + 1;
    const end = Math.min(this.currentPage() * limit, total);
    return `${start} – ${end}`;
  }

  onSaveDocument(index: number, updatedDoc: Record<string, any>): void {
    const sessionId = this.connectionService.sessionId();
    const data = this.collectionData();
    const originalDoc = this.localDocuments()[index];

    if (!sessionId || !data || !originalDoc) return;

    const objectId = String(originalDoc['_id'] ?? '');
    if (!objectId) {
      this.errorMessage.set('Document has no _id — cannot update.');
      return;
    }

    // Compute delta
    const changes: Record<string, any> = {};
    for (const key of Object.keys(updatedDoc)) {
      if (key === '_id') continue;
      if (JSON.stringify(updatedDoc[key]) !== JSON.stringify(originalDoc[key])) {
        changes[key] = updatedDoc[key];
      }
    }

    if (Object.keys(changes).length === 0) {
      this.successMessage.set('No changes detected.');
      setTimeout(() => this.successMessage.set(null), 2000);
      return;
    }

    this.isLoading.set(true);
    this.explorerService
      .updateMongoDocument(sessionId, data.database, data.collection, objectId, changes)
      .subscribe({
        next: () => {
          const updatedDocs = [...this.localDocuments()];
          updatedDocs[index] = { ...originalDoc, ...changes };
          this.localDocuments.set(updatedDocs);
          
          this.successMessage.set('Document updated successfully.');
          this.isLoading.set(false);
          setTimeout(() => this.successMessage.set(null), 3000);
        },
        error: (err: any) => {
          this.errorMessage.set(err?.error?.error || 'Update failed');
          this.isLoading.set(false);
        }
      });
  }
}