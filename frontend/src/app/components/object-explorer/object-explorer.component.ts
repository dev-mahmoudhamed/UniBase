import { Component, inject, input, effect, signal, output, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Tree, TreeNodeExpandEvent, TreeNodeSelectEvent } from 'primeng/tree';
import { TooltipModule } from 'primeng/tooltip';
import { ExplorerService, TreeNode } from '../../services/explorer.service';

@Component({
    selector: 'app-object-explorer',
    standalone: true,
    imports: [Tree, FormsModule, TooltipModule],
    templateUrl: './object-explorer.component.html',
    styleUrls: ['./object-explorer.component.scss']
})
export class ObjectExplorerComponent {
    sessionId = input<string | null>(null);

    allTreeNodes = signal<TreeNode[]>([]);
    isLoading = signal(false);
    errorMessage = signal<string | null>(null);
    nodeSelected = output<any>();
    searchQuery = signal('');

    // Sorted & filtered view — does NOT spread/clone nodes, only re-orders references
    treeNodes = computed(() => {
        const nodes = this.allTreeNodes();
        const query = this.searchQuery().trim().toLowerCase();

        // Process nodes: only sort children of 'database' nodes
        const processed = this.processAndSortNodes(nodes, false);

        if (!query) return processed;
        return this.filterNodes(processed, query);
    });

    private explorerService = inject(ExplorerService);

    constructor() {
        effect(() => {
            const sid = this.sessionId();
            if (sid) {
                this.loadRootNodes(sid);
            } else {
                this.allTreeNodes.set([]);
            }
        });
    }

    /**
     * Processes nodes WITHOUT cloning them, and ONLY sorts children
     * if the parent node is of type 'database'.
     */
    private processAndSortNodes(nodes: TreeNode[], shouldSort: boolean): TreeNode[] {
        let result = [...nodes];

        if (shouldSort) {
            result.sort((a, b) => (a.label || '').localeCompare(b.label || ''));
        }

        for (const node of result) {
            if (node.children && node.children.length > 0) {
                const sortChildren = node.type === 'database';
                node.children = this.processAndSortNodes(node.children, sortChildren);
            }
        }
        return result;
    }

    private filterNodes(nodes: TreeNode[], query: string): TreeNode[] {
        const result: TreeNode[] = [];
        for (const node of nodes) {
            const labelMatch = (node.label || '').toLowerCase().includes(query);
            const filteredChildren = node.children ? this.filterNodes(node.children, query) : [];
            if (labelMatch || filteredChildren.length > 0) {
                result.push({
                    ...node,
                    children: filteredChildren.length > 0 ? filteredChildren : node.children,
                    expanded: filteredChildren.length > 0 ? true : node.expanded
                });
            }
        }
        return result;
    }

    private loadRootNodes(sessionId: string): void {
        this.isLoading.set(true);
        this.errorMessage.set(null);
        this.searchQuery.set('');

        this.explorerService.getChildren(sessionId, 'root').subscribe({
            next: (nodes) => {
                this.allTreeNodes.set(nodes);
                this.isLoading.set(false);
            },
            error: (err) => {
                this.errorMessage.set(err?.error?.error || 'Failed to load explorer');
                this.isLoading.set(false);
            }
        });
    }

    private updateNodeInSource(nodes: TreeNode[], targetKey: string | undefined, children: TreeNode[], error?: string): boolean {
        if (!targetKey) return false;
        for (const node of nodes) {
            if (node.key === targetKey) {
                node.loading = false;
                if (error) {
                    node.children = [{
                        key: 'error-' + targetKey,
                        label: error,
                        icon: 'pi pi-exclamation-triangle',
                        leaf: true,
                        type: 'error'
                    }];
                } else {
                    node.children = children;
                }
                return true;
            }
            if (node.children && this.updateNodeInSource(node.children, targetKey, children, error)) {
                return true;
            }
        }
        return false;
    }

    onNodeExpand(event: TreeNodeExpandEvent): void {
        const node = event.node;
        const sessionId = this.sessionId();

        if (!sessionId || !node) return;

        if (node.children && node.children.length > 0) {
            return;
        }

        const nodeType = node.data?.nodeType || node.type;
        if (!nodeType) return;

        node.loading = true;

        const context: Record<string, string> = {};
        if (node.data) {
            Object.keys(node.data).forEach(key => {
                if (key !== 'nodeType') {
                    context[key] = node.data[key];
                }
            });
        }

        this.explorerService.getChildren(sessionId, nodeType, context).subscribe({
            next: (children) => {
                const sourceNodes = this.allTreeNodes();
                this.updateNodeInSource(sourceNodes, node.key, children);
                this.allTreeNodes.set([...sourceNodes]);
            },
            error: (err) => {
                const sourceNodes = this.allTreeNodes();
                this.updateNodeInSource(sourceNodes, node.key, [], err?.error?.error || 'Failed to load');
                this.allTreeNodes.set([...sourceNodes]);
            }
        });
    }

    onNodeSelect(event: TreeNodeSelectEvent): void {
        const node = event.node;
        if (node && !node.leaf) {
            node.expanded = !node.expanded;
            if (node.expanded) {
                this.onNodeExpand({ node } as TreeNodeExpandEvent);
            }
        }
        this.nodeSelected.emit(event);
    }

    clearSearch(): void {
        this.searchQuery.set('');
    }

    /**
     * Builds an HTML tooltip for MongoDB collection nodes using real stats
     * fetched by the backend's collStats command and field sampling.
     */
    getMongoTooltip(node: TreeNode): string {
        if (node.type !== 'collection') return '';

        const stats = node.data?.stats;

        // If no stats were returned (e.g. permissions issue), show a minimal tooltip
        if (!stats || typeof stats !== 'object') {
            return `<div class="mongo-tooltip">
                <div class="tooltip-header"><strong>${node.label}</strong></div>
                <div class="tooltip-row" style="opacity:0.6">Stats unavailable</div>
            </div>`;
        }

        const count = stats['count'] ?? 0;
        const size = this.formatSize(stats['size'] ?? 0);
        const storageSize = this.formatSize(stats['storageSize'] ?? 0);
        const nindexes = stats['nindexes'] ?? 0;
        const fields: string[] = Array.isArray(stats['fields']) ? stats['fields'] : [];

        const rows: string[] = [];

        rows.push(`<div class="tooltip-header">
            <strong>${node.label}</strong> &bull; ${this.formatCount(count)} doc${count !== 1 ? 's' : ''} &bull; ${size}
        </div>`);

        rows.push(`<div class="tooltip-row">Storage: <strong>${storageSize}</strong></div>`);
        rows.push(`<div class="tooltip-row">Indexes: <strong>${nindexes}</strong></div>`);

        if (fields.length > 0) {
            rows.push(`<div class="tooltip-divider"></div>`);
            rows.push(`<div class="tooltip-fields">Sample fields: ${fields.join(', ')}</div>`);
        }

        return `<div class="mongo-tooltip">${rows.join('')}</div>`;
    }

    formatSize(bytes: number): string {
        if (!bytes || bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    formatCount(num: number): string {
        if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + 'M';
        if (num >= 1_000) return (num / 1_000).toFixed(1) + 'K';
        return num.toString();
    }
}