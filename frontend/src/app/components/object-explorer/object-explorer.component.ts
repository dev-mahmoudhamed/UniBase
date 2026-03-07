import { Component, inject, input, effect, signal, output, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Tree, TreeNodeExpandEvent, TreeNodeSelectEvent } from 'primeng/tree';
import { ExplorerService, TreeNode } from '../../services/explorer.service';

@Component({
    selector: 'app-object-explorer',
    standalone: true,
    imports: [Tree, FormsModule],
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
        const sorted = this.sortNodesInPlace(nodes);
        if (!query) return sorted;
        return this.filterNodes(sorted, query);
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
     * Sorts nodes alphabetically WITHOUT cloning them.
     * Returns a new sorted array whose elements are the same object references
     * so mutations made by PrimeNG (e.g. node.expanded, node.children) survive.
     */
    private sortNodesInPlace(nodes: TreeNode[]): TreeNode[] {
        const sorted = [...nodes].sort((a, b) => (a.label || '').localeCompare(b.label || ''));
        for (const node of sorted) {
            if (node.children && node.children.length > 0) {
                node.children = this.sortNodesInPlace(node.children);
            }
        }
        return sorted;
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
        this.nodeSelected.emit(event);
    }

    clearSearch(): void {
        this.searchQuery.set('');
    }
}
