import { Component, inject, input, output, signal, HostListener, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ConnectionService } from '../../services/connection.service';
import { ProviderService } from '../../services/provider.service';
import { ExplorerService, MongoCollectionData } from '../../services/explorer.service';
import { DatabaseConnection, DatabaseProvider } from '../../models/database.model';
import { ObjectExplorerComponent } from '../object-explorer/object-explorer.component';

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [ObjectExplorerComponent, CommonModule, FormsModule],
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.scss']
})
export class SidebarComponent {
  private connectionService = inject(ConnectionService);
  private providerService = inject(ProviderService);
  private explorerService = inject(ExplorerService);

  @ViewChild(ObjectExplorerComponent) explorer?: ObjectExplorerComponent;

  toggleSidebar = output<void>();
  openConnectionDialog = output<void>();
  editConnectionEvent = output<DatabaseConnection>();
  collectionSelected = output<MongoCollectionData>();
  redisKeySelected = output<{ database: string; key: string }>();
  isCollapsed = input(false);

  connections = this.connectionService.connections;
  activeConnection = this.connectionService.activeConnection;
  sessionId = this.connectionService.sessionId;
  isLoadingMetadata = this.connectionService.isLoadingMetadata;

  contextMenuVisible = signal(false);
  contextMenuX = signal(0);
  contextMenuY = signal(0);
  contextMenuConnection = signal<DatabaseConnection | null>(null);

  // Rename Dialog State
  showRenameDialog = signal(false);
  renameValue = signal('');
  renameTitle = signal('');
  private renameTarget: { type: 'connection' | 'node', data: any } | null = null;

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

  getProviderIcon(provider: string): string {
    return this.providerService.getProviderIcon(provider?.toLowerCase());
  }

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
    if (conn) this.editConnectionEvent.emit(conn);
    this.contextMenuVisible.set(false);
  }

  renameConnection(): void {
    const conn = this.contextMenuConnection();
    if (!conn) return;

    // Check if we should rename a node in the explorer instead
    const node = this.isActive(conn) ? this.explorer?.selectedNode() : null;

    if (node) {
      this.renameValue.set(node.label || '');
      this.renameTitle.set(`Rename ${node.type || 'Item'}`);
      this.renameTarget = { type: 'node', data: node };
    } else {
      this.renameValue.set(conn.name);
      this.renameTitle.set('Rename Connection');
      this.renameTarget = { type: 'connection', data: conn };
    }

    this.showRenameDialog.set(true);
    this.contextMenuVisible.set(false);
  }

  confirmRename(): void {
    const newName = this.renameValue().trim();
    if (!this.renameTarget || !newName) {
      this.showRenameDialog.set(false);
      return;
    }

    if (this.renameTarget.type === 'connection') {
      this.connectionService.updateConnection({ ...this.renameTarget.data, name: newName });
    } else if (this.renameTarget.type === 'node') {
      // Logic for renaming a node in the explorer (e.g., collection/table)
      // Note: Backend support for this might be needed for persistent changes
      this.renameTarget.data.label = newName;
      // You might want to trigger a refresh or emit a rename event here
    }

    this.showRenameDialog.set(false);
    this.renameTarget = null;
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
      if (this.isActive(conn) && this.explorer && this.explorer.selectedNode()) {
        this.explorer.refreshSelectedNode();
      } else {
        this.connectionService.setActiveConnection(conn);
      }
    }
    this.contextMenuVisible.set(false);
  }


  onNodeSelected(event: any): void {
    const node = event.node;
    if (!node) return;

    const connection = this.activeConnection();
    if (!connection) return;

    const nodeType = node.data?.nodeType || node.type;

    if (connection.provider === DatabaseProvider.REDIS && nodeType === 'key') {
      const database = node.data?.database || '0';
      const key = node.data?.key;
      if (key) this.redisKeySelected.emit({ database, key });
      return;
    }

    if (connection.provider !== DatabaseProvider.MONGODB) return;

    if (nodeType === 'collection' || nodeType === 'table') {
      const sessionId = this.sessionId();
      if (!sessionId) return;

      const context: Record<string, string> = {};
      if (node.data) {
        Object.keys(node.data).forEach((key: string) => {
          const val = node.data[key];
          if (key !== 'nodeType' && key !== 'stats' && key !== 'tooltipContent' &&
            (typeof val === 'string' || typeof val === 'number')) {
            context[key] = String(val);
          }
        });
      }

      const database = context['database'] || '';
      const collectionName = context['collection'] || node.label || '';

      this.explorerService.getCollectionData(sessionId, collectionName, context).subscribe({
        next: (data: any) => {
          const enriched: MongoCollectionData = {
            collection: collectionName,
            database,
            documents: data.documents ?? [],
            columns: data.columns ?? [],
            count: data.count ?? 0
          };
          this.collectionSelected.emit(enriched);
        },
        error: () => {
          this.collectionSelected.emit({ collection: collectionName, database, documents: [], columns: [], count: 0 });
        }
      });
    }
  }
}