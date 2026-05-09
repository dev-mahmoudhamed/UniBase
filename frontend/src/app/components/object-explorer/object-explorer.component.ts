import { Component, inject, input, effect, signal, output, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Tree, TreeModule, TreeNodeExpandEvent, TreeNodeSelectEvent } from 'primeng/tree';
import { CommonModule } from '@angular/common';
import { ExplorerService, TreeNode } from '../../services/explorer.service';

@Component({
    selector: 'app-object-explorer',
    standalone: true,
    imports: [FormsModule, CommonModule, TreeModule],
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
    selectedNode = signal<TreeNode | null>(null);

    tooltip = signal<TooltipState>({
        visible: false,
        x: 0,
        y: 0,
        loading: false,
        content: null,
        nodeKey: null
    });

    private tooltipHideTimer: ReturnType<typeof setTimeout> | null = null;
    private fetchedKeys = new Set<string>();

    treeNodes = computed(() => {
        const nodes = this.allTreeNodes();
        const query = this.searchQuery().trim().toLowerCase();
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
                this.fetchedKeys.clear();
            }
        });
    }

    private processAndSortNodes(nodes: TreeNode[], shouldSort: boolean): TreeNode[] {
        let result = [...nodes];
        if (shouldSort) {
            result.sort((a, b) => (a.label || '').localeCompare(b.label || ''));
        }
        for (const node of result) {
            if (node.children && node.children.length > 0) {
                node.children = this.processAndSortNodes(node.children, node.type === 'database');
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
        this.fetchedKeys.clear();

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

    private updateNodeInSource(
        nodes: TreeNode[],
        targetKey: string | undefined,
        children: TreeNode[],
        error?: string
    ): boolean {
        if (!targetKey) return false;
        for (const node of nodes) {
            if (node.key === targetKey) {
                node.loading = false;
                node.children = error
                    ? [{ key: 'error-' + targetKey, label: error, icon: 'pi pi-exclamation-triangle', leaf: true, type: 'error' }]
                    : children;
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
        if (node.children && node.children.length > 0) return;

        const nodeType = node.data?.nodeType || node.type;
        if (!nodeType) return;

        node.loading = true;

        const context: Record<string, string> = {};
        if (node.data) {
            Object.keys(node.data).forEach(key => {
                if (key !== 'nodeType' && key !== 'stats') {
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
        this.selectedNode.set(node);
        if (node && !node.leaf) {
            node.expanded = !node.expanded;
            if (node.expanded && (!node.children || node.children.length === 0)) {
                this.onNodeExpand({ node } as TreeNodeExpandEvent);
            }
        }
        this.nodeSelected.emit(event);
    }

    onNodeContextMenu(event: any): void {
        const node = event.node;
        if (node) {
            this.selectedNode.set(node);
            this.nodeSelected.emit(event);
        }
    }

    refreshSelectedNode(): void {

        const node = this.selectedNode();
        console.log("selected not ->", node);

        if (!node) return;

        if (!node.leaf) {
            node.children = [];
            node.expanded = false;
        }

        this.onNodeSelect({ node } as TreeNodeSelectEvent);
    }

    clearSearch(): void {
        this.searchQuery.set('');
    }

    onNodeMouseEnter(event: MouseEvent, node: TreeNode): void {
        const nodeType = node.data?.nodeType || node.type;
        if (nodeType !== 'collection') return;

        if (this.tooltipHideTimer) {
            clearTimeout(this.tooltipHideTimer);
            this.tooltipHideTimer = null;
        }

        const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();

        this.tooltip.set({
            visible: true,
            x: rect.right + window.scrollX + 8,
            y: rect.top + window.scrollY,
            loading: !this.fetchedKeys.has(node.key || ''),
            content: node.data?.tooltipContent ?? null,
            nodeKey: node.key ?? null
        });

        if (!this.fetchedKeys.has(node.key || '')) {
            this.fetchTooltipData(node);
        }
    }

    onNodeMouseLeave(): void {
        this.tooltipHideTimer = setTimeout(() => {
            this.tooltip.update(t => ({ ...t, visible: false }));
        }, 150);
    }

    private fetchTooltipData(node: TreeNode): void {
        const sessionId = this.sessionId();
        if (!sessionId || !node.data?.collection) return;

        this.explorerService
            .getCollectionMetadata(sessionId, node.data['collection'], { database: node.data['database'] })
            .subscribe({
                next: (response) => {
                    const meta = response?.data?.metadata;
                    if (!meta) return;

                    const content: TooltipContent = {
                        name: node.label || '',
                        count: meta.count ?? 0,
                        size: meta.size ?? 0,
                        storageSize: meta.storageSize ?? 0,
                        indexCount: meta.indexCount ?? 0,
                        unusedIndices: meta.unusedIndices ?? 0,
                        avgQueryTime: meta.avgQueryTime ?? 'N/A',
                        readsPerSec: meta.readsPerSec ?? 0,
                        writesPerSec: meta.writesPerSec ?? 0,
                        trend: meta.trend ?? '',
                        fields: Array.isArray(meta.fields) ? meta.fields : [],
                        alerts: Array.isArray(meta.alerts) ? meta.alerts : []
                    };

                    node.data['tooltipContent'] = content;
                    this.fetchedKeys.add(node.key || '');

                    this.tooltip.update(t =>
                        t.nodeKey === node.key ? { ...t, loading: false, content } : t
                    );

                    this.allTreeNodes.set([...this.allTreeNodes()]);
                },
                error: () => {
                    this.fetchedKeys.add(node.key || '');
                    this.tooltip.update(t =>
                        t.nodeKey === node.key ? { ...t, loading: false } : t
                    );
                }
            });
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


interface TooltipState {
    visible: boolean;
    x: number;
    y: number;
    loading: boolean;
    content: TooltipContent | null;
    nodeKey: string | null;
}

interface TooltipContent {
    name: string;
    count: number;
    size: number;
    storageSize: number;
    indexCount: number;
    unusedIndices: number;
    avgQueryTime: string;
    readsPerSec: number;
    writesPerSec: number;
    trend: string;
    fields: string[];
    alerts: string[];
}
