import { Component, inject, input, output, signal, HostListener } from '@angular/core';
import { ConnectionService } from '../../services/connection.service';
import { ProviderService } from '../../services/provider.service';
import { ExplorerService } from '../../services/explorer.service';
import { DatabaseConnection, DatabaseProvider } from '../../models/database.model';
import { ObjectExplorerComponent } from '../object-explorer/object-explorer.component';

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [ObjectExplorerComponent],
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.scss']
})
export class SidebarComponent {
  private connectionService = inject(ConnectionService);
  private providerService = inject(ProviderService);
  private explorerService = inject(ExplorerService);

  toggleSidebar = output<void>();
  openConnectionDialog = output<void>();
  editConnectionEvent = output<DatabaseConnection>();
  collectionSelected = output<any>();
  redisKeySelected = output<{ database: string; key: string }>();
  isCollapsed = input(false);

  // Signals
  connections = this.connectionService.connections;
  activeConnection = this.connectionService.activeConnection;

  // Session & Metadata State
  sessionId = this.connectionService.sessionId;
  isLoadingMetadata = this.connectionService.isLoadingMetadata;

  // Context menu state
  contextMenuVisible = signal(false);
  contextMenuX = signal(0);
  contextMenuY = signal(0);
  contextMenuConnection = signal<DatabaseConnection | null>(null);



  @HostListener('document:click')
  onDocumentClick(): void {
    this.contextMenuVisible.set(false);
  }

  onAddConnection(): void {
    this.openConnectionDialog.emit();
  }

  onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }

  selectConnection(connection: DatabaseConnection): void {
    if (this.isActive(connection)) {
      this.connectionService.setActiveConnection(null);
    } else {
      this.connectionService.setActiveConnection(connection);
    }
  }

  refreshConnection(event: Event, connection: DatabaseConnection): void {
    event.stopPropagation();
    this.connectionService.setActiveConnection(connection);
  }

  deleteConnection(event: Event, id: string): void {
    event.stopPropagation();
    if (confirm('Are you sure you want to delete this connection?')) {
      this.connectionService.removeConnection(id);
    }
  }

  isActive(connection: DatabaseConnection): boolean {
    return this.activeConnection()?.id === connection.id;
  }

  isMongo(connection: DatabaseConnection): boolean {
    return connection.provider === DatabaseProvider.MONGODB;
  }

  isRedis(connection: DatabaseConnection): boolean {
    return connection.provider === DatabaseProvider.REDIS;
  }

  getProviderIcon(provider: string): string {
    return this.providerService.getProviderIcon(provider?.toLowerCase());
  }

  // Context menu
  onRightClick(event: MouseEvent, connection: DatabaseConnection): void {
    event.preventDefault();
    event.stopPropagation();
    this.contextMenuX.set(event.clientX);
    this.contextMenuY.set(event.clientY);
    this.contextMenuConnection.set(connection);
    this.contextMenuVisible.set(true);
  }

  editConnection(): void {
    const conn = this.contextMenuConnection();
    if (conn) {
      this.editConnectionEvent.emit(conn);
    }
    this.contextMenuVisible.set(false);
  }

  deleteFromContextMenu(): void {
    const conn = this.contextMenuConnection();
    if (conn && confirm('Are you sure you want to delete this connection?')) {
      this.connectionService.removeConnection(conn.id);
    }
    this.contextMenuVisible.set(false);
  }

  refreshFromContextMenu(): void {
    const conn = this.contextMenuConnection();
    if (conn) {
      this.connectionService.setActiveConnection(conn);
    }
    this.contextMenuVisible.set(false);
  }

  // Handle node click from object explorer
  onNodeSelected(event: any): void {
    const node = event.node;
    if (!node) return;

    const connection = this.activeConnection();
    if (!connection) return;

    const nodeType = node.data?.nodeType || node.type;

    // Redis key selected
    if (connection.provider === DatabaseProvider.REDIS && nodeType === 'key') {
      const database = node.data?.database || '0';
      const key = node.data?.key;
      if (key) {
        this.redisKeySelected.emit({ database, key });
      }
      return;
    }

    // MongoDB collection selected
    if (connection.provider !== DatabaseProvider.MONGODB) return;

    if (nodeType === 'collection' || nodeType === 'table') {
      const sessionId = this.sessionId();
      if (!sessionId) return;

      const context: Record<string, string> = {};
      if (node.data) {
        Object.keys(node.data).forEach((key: string) => {
          if (key !== 'nodeType') {
            context[key] = node.data[key];
          }
        });
      }

      this.explorerService.getCollectionData(sessionId, node.label || '', context).subscribe({
        next: (data: any) => {
          this.collectionSelected.emit(data);
        },
        error: (err: any) => {
          this.collectionSelected.emit({ error: err?.error?.error || 'Failed to fetch collection data' });
        }
      });
    }
  }
}