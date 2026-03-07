import { Component, signal, inject, computed } from '@angular/core';
import { EMPTY, timer, of } from 'rxjs';
import { expand, filter, last, concatMap, catchError } from 'rxjs/operators';
import { AngularSplitModule } from 'angular-split';
import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { QueryEditorComponent } from './components/query-editor/query-editor.component';
import { ResultsViewerComponent } from './components/results-viewer/results-viewer.component';
import { ConnectionDialogComponent } from './components/connection-dialog/connection-dialog.component';
import { RedisViewerComponent } from './components/redis-viewer/redis-viewer.component';
import { QueryService } from './services/query.service';
import { ConnectionService } from './services/connection.service';
import { DatabaseConnection, DatabaseProvider, QueryResult, QueryError } from './models/database.model';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    AngularSplitModule,
    FormsModule,
    MonacoEditorModule,
    SidebarComponent,
    QueryEditorComponent,
    ResultsViewerComponent,
    ConnectionDialogComponent,
    RedisViewerComponent
  ],
  templateUrl: './app.html',
  styleUrls: ['./app.scss']
})
export class AppComponent {
  sidebarCollapsed = signal(false);
  readonly SIDEBAR_WIDTH = '260px';
  readonly SIDEBAR_COLLAPSED_WIDTH = '44px';
  showConnectionDialog = signal(false);
  editingConnection = signal<DatabaseConnection | null>(null);
  queryResult = signal<QueryResult | QueryError | null>(null);
  isLoadingQuery = signal(false);
  jsonViewerData = signal<string>('');
  selectedRedisKey = signal<{ database: string; key: string } | null>(null);

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

  jsonEditorOptions = {
    theme: 'json-viewer-dark',
    language: 'json',
    minimap: { enabled: false },
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace",
    fontLigatures: true,
    automaticLayout: true,
    scrollBeyondLastLine: false,
    lineNumbers: 'on' as 'on',
    renderLineHighlight: 'all' as 'all',
    tabSize: 2,
    readOnly: true,
    cursorBlinking: 'smooth' as 'smooth',
    cursorSmoothCaretAnimation: 'on' as 'on',
    smoothScrolling: true,
    padding: { top: 16, bottom: 16 },
    wordWrap: 'on' as 'on',
    folding: true,
    links: true,
    roundedSelection: true,
    letterSpacing: 0.5,
    lineHeight: 20,
    scrollbar: {
      vertical: 'visible' as 'visible',
      horizontal: 'visible' as 'visible',
      useShadows: false,
      verticalScrollbarSize: 10,
      horizontalScrollbarSize: 10
    }
  };

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

  onCollectionSelected(data: any): void {
    try {
      this.jsonViewerData.set(JSON.stringify(data, null, 2));
    } catch {
      this.jsonViewerData.set('// Error parsing collection data');
    }
  }

  onRedisKeySelected(event: { database: string; key: string }): void {
    this.selectedRedisKey.set({ ...event });
  }

  onJsonEditorInit(editor: any): void {
    const monaco = (window as any).monaco;
    if (monaco) {
      monaco.editor.defineTheme('json-viewer-dark', {
        base: 'vs-dark',
        inherit: true,
        rules: [
          { token: 'string.key.json', foreground: '56b6c2' },
          { token: 'string.value.json', foreground: '98c379' },
          { token: 'number', foreground: 'd19a66' },
          { token: 'keyword.json', foreground: 'c678dd' },
          { token: 'delimiter', foreground: 'abb2bf' },
        ],
        colors: {
          'editor.background': '#161a1e',
          'editor.foreground': '#abb2bf',
          'editor.lineHighlightBackground': '#1e2429',
          'editor.selectionBackground': '#2c3e50',
          'editorCursor.foreground': '#56b6c2',
          'editorWhitespace.foreground': '#3b4048',
          'editorIndentGuide.background': '#2c313a',
          'editorIndentGuide.activeBackground': '#56b6c2'
        }
      });
      monaco.editor.setTheme('json-viewer-dark');
    }
  }

  executeQuery(event: { query: string; queryId: string; sessionId: string }): void {
    const { query, queryId, sessionId } = event;

    this.isLoadingQuery.set(true);
    this.queryResult.set(null);

    this.queryService.executeQuery(sessionId, queryId, query).pipe(
      // Move polling logic here
      expand(response => {
        if (response?.message === 'query timeout') {
          return timer(2000).pipe(
            concatMap(() => this.queryService.executeQuery(sessionId, queryId, query).pipe(
              catchError(err => of({ status: 'error', error: err?.message || 'Polling failed' } as any))
            ))
          );
        }
        return EMPTY;
      }),
      filter(response => response?.message !== 'query timeout'),
      last()
    ).subscribe({
      next: (response) => {
        if (!response) {
          this.queryResult.set({
            message: 'No response from server'
          });
        } else if (response.status === 'success' && response.results) {
          this.queryResult.set(response.results);
        } else if (response.error) {
          this.queryResult.set({
            message: response.error
          });
        } else {
          this.queryResult.set({
            message: response.message || 'Unknown response from server'
          });
        }
        this.isLoadingQuery.set(false);
      },
      error: (err) => {
        const errorMsg = err?.error?.error || err?.message || 'An error occurred while executing the query';
        this.queryResult.set({
          message: errorMsg
        });
        this.isLoadingQuery.set(false);
      }
    });
  }

  testConnection(): void {
    const activeConnection = this.connectionService.activeConnection();

    if (!activeConnection) {
      return;
    }

    this.isLoadingQuery.set(true);
    this.queryResult.set(null);

    this.queryService.testConnection(activeConnection).subscribe({
      next: (result) => {
        this.queryResult.set(result);
        this.isLoadingQuery.set(false);
      },
      error: (err) => {
        const errorMsg = err?.error?.error || err?.message || 'An error occurred while testing the connection';
        this.queryResult.set({
          message: errorMsg
        });
        this.isLoadingQuery.set(false);
      }
    });
  }
}