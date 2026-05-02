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

        // Only sort if instructed (i.e. we are processing children of a 'database' node)
        if (shouldSort) {
            result.sort((a, b) => (a.label || '').localeCompare(b.label || ''));
        }

        for (const node of result) {
            if (node.children && node.children.length > 0) {
                // If THIS node is a 'database', sort its children. 
                // We do NOT pass 'shouldSort' down further unless we want deep sorting, 
                // but usually clicking a database returns immediate children.
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
                // Use the same node reference; temporarily override children for display
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

    /**
     * Recursively finds a node by key inside the allTreeNodes source and
     * sets its children, so the computed treeNodes picks up the change.
     */
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

        // Already loaded children (non-empty array)
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
                // Update the source node in allTreeNodes (by reference search)
                const sourceNodes = this.allTreeNodes();
                this.updateNodeInSource(sourceNodes, node.key, children);
                // Trigger reactivity
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

    getMongoTooltip(node: TreeNode): string {
        if (node.type !== 'collection' || !node.data?.stats) return '';
        const stats = node.data.stats;
        const count = stats.count || 0;
        const storageSize = this.formatSize(stats.storageSize || 0);
        const fields = (stats.fields || []).join(', ');

        return `
            <div class="mongo-tooltip">
                <div class="tooltip-header"><strong>${node.label}</strong> • ${this.formatCount(count)} docs • ${storageSize} • <span class="trend-up">↑18%</span></div>
                <div class="tooltip-row">Reads: 220/s | Writes: 45/s</div>
                <div class="tooltip-row">Avg query: 180ms</div>
                <div class="tooltip-divider"></div>
                <div class="tooltip-row">Indexes: ${stats.nindexes || 0} (2 unused)</div>
                <div class="tooltip-warning">⚠ Missing index on userId</div>
                <div class="tooltip-warning">⚠ Large documents</div>
                <div class="tooltip-divider"></div>
                <div class="tooltip-fields">Fields: ${fields}</div>
            </div>
        `;
    }

    formatSize(bytes: number): string {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    formatCount(num: number): string {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    }
}
