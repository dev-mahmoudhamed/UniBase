import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { ApiResponse } from '../models/database.model';


export interface ExplorerNode {
    key: string;
    label: string;
    type: string;
    icon: string;
    leaf: boolean;
    data?: Record<string, any>;
    children?: ExplorerNode[];
}

export interface ExplorerNodeResponse {
    nodes: ExplorerNode[];
}

export interface TreeNode {
    key?: string;
    label?: string;
    icon?: string;
    leaf?: boolean;
    data?: any;
    children?: TreeNode[];
    loading?: boolean;
    expanded?: boolean;
    type?: string;
    parent?: TreeNode;
}

export interface MongoCollectionData {
    collection: string;
    database: string;
    documents: Record<string, any>[];
    columns: string[];
    count: number;
}
 
@Injectable({
    providedIn: 'root'
})
export class ExplorerService {
    private baseUrl = '/api/explorer';

    /** Key of a tree node whose children changed (e.g. a Redis key was renamed); the explorer reloads it. */
    readonly staleNode = signal<string | null>(null);
    private mongoBaseUrl = '/api/mongo';
 
    constructor(private http: HttpClient) { }
 
    getChildren(
        sessionId: string,
        nodeType: string,
        context?: Record<string, string>
    ): Observable<TreeNode[]> {
        return this.http
            .post<any>(`${this.baseUrl}/children`, {
                sessionId,
                nodeType,
                context: context || {}
            })
            .pipe(map(response => {
                const nodes = response?.data?.nodes || [];
                return this.mapToTreeNodes(nodes);
            }));
    }
 
    getCollectionData(
        sessionId: string,
        collectionName: string,
        context?: Record<string, string>,
        options?: {
            filter?: string;
            projection?: string;
            sort?: string;
            skip?: number;
            limit?: number;
        }
    ): Observable<any> {
        return this.http.post<any>(`${this.baseUrl}/collection-data`, {
            sessionId,
            collectionName,
            context: context || {},
            filter: options?.filter || '',
            projection: options?.projection || '',
            sort: options?.sort || '',
            skip: options?.skip || 0,
            limit: options?.limit || 20
        }).pipe(map(response => response?.data));
    }

    getCollectionMetadata(
        sessionId: string,
        collectionName: string,
        context?: Record<string, string>
    ): Observable<any> {
        return this.http.post<any>(`${this.baseUrl}/collection-metadata`, {
            sessionId,
            collectionName,
            context: context || {}
        });
    }



    /**
     * Saves changed top-level fields. `id` and `changes` stay in the relaxed
     * Extended JSON form the server sent (e.g. {"$oid": "..."}), so types round-trip.
     */
    updateMongoDocument(
        sessionId: string,
        database: string,
        collection: string,
        id: unknown,
        changes: Record<string, unknown>
    ): Observable<{ message: string }> {
        return this.http.post<{ message: string }>(`${this.mongoBaseUrl}/document/update`, {
            sessionId,
            database,
            collection,
            id,
            changes
        });
    }

    private mapToTreeNodes(nodes: ExplorerNode[]): TreeNode[] {
        return nodes.map(node => {
            const treeNode: TreeNode = {
                key: node.key,
                label: node.label,
                icon: node.icon,
                leaf: node.leaf,
                data: { ...node.data, nodeType: node.type },
                type: node.type
            };

            if (node.children && node.children.length > 0) {
                treeNode.children = this.mapToTreeNodes(node.children);
                treeNode.leaf = false;
            } else if (!node.leaf) {
                treeNode.children = [];
                treeNode.leaf = false;
            }

            return treeNode;
        });
    }
}