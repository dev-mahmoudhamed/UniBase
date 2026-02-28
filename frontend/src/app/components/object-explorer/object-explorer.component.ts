import { Component, inject, input, effect, signal, output } from '@angular/core';

import { Tree, TreeNodeExpandEvent, TreeNodeSelectEvent } from 'primeng/tree';
import { ExplorerService, TreeNode } from '../../services/explorer.service';

@Component({
    selector: 'app-object-explorer',
    standalone: true,
    imports: [Tree],
    templateUrl: './object-explorer.component.html',
    styleUrls: ['./object-explorer.component.scss']
})
export class ObjectExplorerComponent {
    sessionId = input<string | null>(null);

    treeNodes = signal<TreeNode[]>([]);
    isLoading = signal(false);
    errorMessage = signal<string | null>(null);
    nodeSelected = output<any>();

    private explorerService = inject(ExplorerService);

    constructor() {
        // React to sessionId changes - load root nodes when a new session is established
        effect(() => {
            const sid = this.sessionId();
            if (sid) {
                this.loadRootNodes(sid);
            } else {
                this.treeNodes.set([]);
            }
        });
    }

    private loadRootNodes(sessionId: string): void {
        this.isLoading.set(true);
        this.errorMessage.set(null);

        this.explorerService.getChildren(sessionId, 'root').subscribe({
            next: (nodes) => {
                this.treeNodes.set(nodes);
                this.isLoading.set(false);
            },
            error: (err) => {
                this.errorMessage.set(err?.error?.error || 'Failed to load explorer');
                this.isLoading.set(false);
            }
        });
    }

    onNodeExpand(event: TreeNodeExpandEvent): void {
        const node = event.node;
        const sessionId = this.sessionId();

        if (!sessionId || !node) return;

        // Already loaded children (non-empty array)
        if (node.children && node.children.length > 0) {
            return;
        }

        // Get the node type for the API call
        const nodeType = node.data?.nodeType || node.type;
        if (!nodeType) return;

        node.loading = true;

        // Build context from node data (remove nodeType as it's not context)
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
                node.children = children;
                node.loading = false;

                // Update the tree to reflect changes
                this.treeNodes.set([...this.treeNodes()]);
            },
            error: (err) => {
                node.loading = false;
                node.children = [{
                    key: 'error',
                    label: err?.error?.error || 'Failed to load',
                    icon: 'pi pi-exclamation-triangle',
                    leaf: true,
                    type: 'error'
                }];
                this.treeNodes.set([...this.treeNodes()]);
            }
        });
    }

    onNodeSelect(event: TreeNodeSelectEvent): void {
        this.nodeSelected.emit(event);
    }
}
