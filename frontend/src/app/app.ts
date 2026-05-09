import { Component, signal, inject, computed } from '@angular/core';
import { EMPTY, timer, of } from 'rxjs';
import { expand, filter, last, concatMap, catchError } from 'rxjs/operators';
import { AngularSplitModule } from 'angular-split';
import { FormsModule } from '@angular/forms';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { QueryEditorComponent } from './components/query-editor/query-editor.component';
import { ResultsViewerComponent } from './components/results-viewer/results-viewer.component';
import { ConnectionDialogComponent } from './components/connection-dialog/connection-dialog.component';
import { RedisViewerComponent } from './components/redis-viewer/redis-viewer.component';
import { MongoViewerComponent } from './components/mongo-viewer/mongo-viewer.component';
import { QueryService } from './services/query.service';
import { ConnectionService } from './services/connection.service';
import { MongoCollectionData } from './services/explorer.service';
import { hashQueryId } from './utils/hash.utils';
import { DatabaseConnection, DatabaseProvider, QueryResult, QueryError } from './models/database.model';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    AngularSplitModule,
    FormsModule,
    SidebarComponent,
    QueryEditorComponent,
    ResultsViewerComponent,
    ConnectionDialogComponent,
    RedisViewerComponent,
    MongoViewerComponent
  ],

  templateUrl: './app.html',
  styleUrls: ['./app.scss']
})
export class AppComponent {
  sidebarCollapsed = signal(false);
  showConnectionDialog = signal(false);
  editingConnection = signal<DatabaseConnection | null>(null);
  queryResult = signal<QueryResult | QueryError | null>(null);
  isLoadingQuery = signal(false);

  selectedCollectionData = signal<MongoCollectionData | null>(null);

  selectedRedisKey = signal<{ database: string; key: string } | null>(null);
  currentQueryText = signal<string>('');
  currentLanguage = computed(() => this.isMongoConnection() ? 'javascript' : 'sql');

  private queryService = inject(QueryService);
  private connectionService = inject(ConnectionService);

  isMongoConnection = computed(() => {
    const conn = this.connectionService.activeConnection();
    return conn?.provider === DatabaseProvider.MONGODB;
  });

  isRedisConnection = computed(() => {
    const conn = this.connectionService.activeConnection();
    return conn?.provider === DatabaseProvider.REDIS;
  });

  toggleSidebar(): void {
    this.sidebarCollapsed.update(v => !v);
  }

  openConnectionDialog(): void {
    this.editingConnection.set(null);
    this.showConnectionDialog.set(true);
  }

  editConnection(connection: DatabaseConnection): void {
    this.editingConnection.set(connection);
    this.showConnectionDialog.set(true);
  }

  closeConnectionDialog(): void {
    this.showConnectionDialog.set(false);
    this.editingConnection.set(null);
  }

  onCollectionSelected(data: MongoCollectionData): void {
    this.selectedCollectionData.set(data);

    // if (this.isMongoConnection()) {
    //   const defaultQuery = `db.${data.collection}.find()`;
    //   this.currentQueryText.set(defaultQuery);

    //   const sessionId = this.connectionService.sessionId();
    //   if (sessionId) {
    //     this.executeQuery({
    //       query: defaultQuery,
    //       queryId: hashQueryId(defaultQuery),
    //       sessionId
    //     });
    //   }
    // }
  }

  onRedisKeySelected(event: { database: string; key: string }): void {
    this.selectedRedisKey.set({ ...event });
  }

  executeQuery(event: { query: string; queryId: string; sessionId: string }): void {
    const { query, queryId, sessionId } = event;

    this.isLoadingQuery.set(true);
    this.queryResult.set(null);

    this.queryService.executeQuery(sessionId, queryId, query).pipe(
      expand(response => {
        if (response?.message === 'query timeout' || response?.status === 'accepted') {
          return timer(2000).pipe(
            concatMap(() => this.queryService.executeQuery(sessionId, queryId, query).pipe(
              catchError(err => of({ status: 'error', error: err?.message || 'Polling failed' } as any))
            ))
          );
        }
        return EMPTY;
      }),
      filter(response => response?.message !== 'query timeout' && response?.status !== 'accepted'),
      last()
    ).subscribe({
      next: (response) => {
        if (!response) {
          this.queryResult.set({ message: 'No response from server' });
        } else if (response.status === 'success' && response.data?.results) {
          this.queryResult.set(response.data.results);
        } else if (response.error) {
          this.queryResult.set({ message: response.error });
        } else {
          this.queryResult.set({ message: response.message || 'Unknown response from server' });
        }
        this.isLoadingQuery.set(false);
      },
      error: (err) => {
        const errorMsg = err?.error?.error || err?.message || 'An error occurred while executing the query';
        this.queryResult.set({ message: errorMsg });
        this.isLoadingQuery.set(false);
      }
    });
  }

}