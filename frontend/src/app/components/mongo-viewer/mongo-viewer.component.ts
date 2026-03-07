import { Component, inject, input, signal, computed, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import { ExplorerService, MongoCollectionData } from '../../services/explorer.service';
import { ConnectionService } from '../../services/connection.service';

@Component({
  selector: 'app-mongo-viewer',
  standalone: true,
  imports: [FormsModule, MonacoEditorModule],
  templateUrl: './mongo-viewer.component.html',
  styleUrls: ['./mongo-viewer.component.scss']
})
export class MongoViewerComponent {
  collectionData = input<MongoCollectionData | null>(null);

  private explorerService = inject(ExplorerService);
  private connectionService = inject(ConnectionService);

  // Local mutable copy of documents (updated after successful saves)
  localDocuments = signal<Record<string, any>[]>([]);

  selectedDocIndex = signal<number | null>(null);
  editedJson = signal<string>('');

  isSaving = signal(false);
  errorMessage = signal<string | null>(null);
  successMessage = signal<string | null>(null);

  // ── Computed helpers ────────────────────────────────────────────────────────

  selectedDoc = computed(() => {
    const idx = this.selectedDocIndex();
    if (idx === null) return null;
    return this.localDocuments()[idx] ?? null;
  });

  isValidJson = computed(() => {
    const str = this.editedJson();
    if (!str.trim()) return false;
    try {
      JSON.parse(str);
      return true;
    } catch {
      return false;
    }
  });

  isDirty = computed(() => {
    const doc = this.selectedDoc();
    if (!doc) return false;
    if (!this.isValidJson()) return true; // still dirty (broken JSON)
    try {
      const edited = JSON.parse(this.editedJson());
      return JSON.stringify(edited) !== JSON.stringify(doc);
    } catch {
      return true;
    }
  });

  changedFieldCount = computed(() => {
    const doc = this.selectedDoc();
    if (!doc || !this.isValidJson() || !this.isDirty()) return 0;
    try {
      const edited = JSON.parse(this.editedJson());
      return Object.keys(edited).filter(
        k => k !== '_id' && JSON.stringify(edited[k]) !== JSON.stringify(doc[k])
      ).length;
    } catch {
      return 0;
    }
  });

  // ── Effects ─────────────────────────────────────────────────────────────────

  constructor() {
    // When a new collection is loaded, reset everything
    effect(() => {
      const data = this.collectionData();
      this.localDocuments.set(data?.documents ? [...data.documents] : []);
      this.selectedDocIndex.set(null);
      this.editedJson.set('');
      this.errorMessage.set(null);
      this.successMessage.set(null);
    });

    // When a document is selected, populate the editor
    effect(() => {
      const idx = this.selectedDocIndex();
      const docs = this.localDocuments();
      if (idx !== null && docs[idx]) {
        this.editedJson.set(JSON.stringify(docs[idx], null, 2));
        this.errorMessage.set(null);
        this.successMessage.set(null);
      } else if (idx === null) {
        this.editedJson.set('');
      }
    });
  }

  // ── User actions ────────────────────────────────────────────────────────────

  selectDocument(index: number): void {
    if (this.isDirty() && this.selectedDocIndex() !== null) {
      if (!confirm('You have unsaved changes. Discard and switch document?')) return;
    }
    this.selectedDocIndex.set(index);
  }

  saveChanges(): void {
    const sessionId = this.connectionService.sessionId();
    const data = this.collectionData();
    const doc = this.selectedDoc();
    const idx = this.selectedDocIndex();

    if (!sessionId || !data || !doc || idx === null) return;

    if (!this.isValidJson()) {
      this.errorMessage.set('Cannot save — JSON is invalid. Fix syntax errors first.');
      return;
    }

    let edited: Record<string, any>;
    try {
      edited = JSON.parse(this.editedJson());
    } catch {
      this.errorMessage.set('Cannot parse JSON.');
      return;
    }

    // Compute the delta: only changed or newly added fields (skip _id)
    const changes: Record<string, any> = {};
    for (const key of Object.keys(edited)) {
      if (key === '_id') continue;
      if (JSON.stringify(edited[key]) !== JSON.stringify(doc[key])) {
        changes[key] = edited[key];
      }
    }

    if (Object.keys(changes).length === 0) {
      this.successMessage.set('No changes detected.');
      setTimeout(() => this.successMessage.set(null), 2500);
      return;
    }

    const objectId = String(doc['_id'] ?? '');
    if (!objectId) {
      this.errorMessage.set('Document has no _id — cannot update.');
      return;
    }

    this.isSaving.set(true);
    this.errorMessage.set(null);
    this.successMessage.set(null);

    this.explorerService
      .updateMongoDocument(sessionId, data.database, data.collection, objectId, changes)
      .subscribe({
        next: () => {
          // Merge changes back into the local copy so isDirty becomes false
          const updatedDoc = { ...doc, ...changes };
          const updatedDocs = [...this.localDocuments()];
          updatedDocs[idx] = updatedDoc;
          this.localDocuments.set(updatedDocs);

          const fieldWord = Object.keys(changes).length === 1 ? 'field' : 'fields';
          this.successMessage.set(
            `Saved — ${Object.keys(changes).length} ${fieldWord} updated.`
          );
          this.isSaving.set(false);
          setTimeout(() => this.successMessage.set(null), 3000);
        },
        error: (err: any) => {
          this.errorMessage.set(err?.error?.error || 'Save failed. Please try again.');
          this.isSaving.set(false);
        }
      });
  }

  resetChanges(): void {
    const doc = this.selectedDoc();
    if (!doc) return;
    this.editedJson.set(JSON.stringify(doc, null, 2));
    this.errorMessage.set(null);
    this.successMessage.set(null);
  }

  formatJson(): void {
    if (!this.isValidJson()) return;
    try {
      const parsed = JSON.parse(this.editedJson());
      this.editedJson.set(JSON.stringify(parsed, null, 2));
    } catch { }
  }

  // ── Display helpers ─────────────────────────────────────────────────────────

  getDocumentLabel(doc: Record<string, any>, index: number): string {
    const id = doc['_id'];
    if (!id) return `Document ${index + 1}`;
    const str = String(id);
    // Show first 20 chars of the ObjectID hex
    return str.length > 20 ? str.substring(0, 20) + '…' : str;
  }

  getDocumentPreview(doc: Record<string, any>): string {
    const keys = Object.keys(doc).filter(k => k !== '_id');
    if (keys.length === 0) return '(empty)';
    const first = keys[0];
    const val = doc[first];
    const preview = typeof val === 'object' ? '{…}' : String(val).substring(0, 24);
    return `${first}: ${preview}`;
  }

  // ── Monaco editor ───────────────────────────────────────────────────────────

  editorOptions = {
    theme: 'mongo-editor-dark',
    language: 'json',
    minimap: { enabled: false },
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace",
    fontLigatures: true,
    automaticLayout: true,
    scrollBeyondLastLine: false,
    lineNumbers: 'on' as const,
    renderLineHighlight: 'all' as const,
    tabSize: 2,
    readOnly: false,
    cursorBlinking: 'smooth' as const,
    smoothScrolling: true,
    padding: { top: 12, bottom: 12 },
    wordWrap: 'on' as const,
    folding: true,
    roundedSelection: true,
    letterSpacing: 0.5,
    lineHeight: 20,
    scrollbar: {
      vertical: 'visible' as const,
      horizontal: 'visible' as const,
      useShadows: false,
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8
    }
  };

  onEditorInit(editor: any): void {
    const monaco = (window as any).monaco;
    if (!monaco) return;

    monaco.editor.defineTheme('mongo-editor-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { token: 'string.key.json', foreground: '56b6c2' },
        { token: 'string.value.json', foreground: '98c379' },
        { token: 'number', foreground: 'd19a66' },
        { token: 'keyword.json', foreground: 'c678dd' },
        { token: 'delimiter', foreground: 'abb2bf' }
      ],
      colors: {
        'editor.background': '#161a1e',
        'editor.foreground': '#abb2bf',
        'editor.lineHighlightBackground': '#1e2429',
        'editor.selectionBackground': '#2c3e50',
        'editorCursor.foreground': '#56b6c2',
        'editorWhitespace.foreground': '#3b4048',
        'editorIndentGuide.background1': '#2c313a',
        'editorIndentGuide.activeBackground1': '#56b6c2'
      }
    });
    monaco.editor.setTheme('mongo-editor-dark');

    // Ctrl+S shortcut to save
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS,
      () => {
        if (this.isDirty() && !this.isSaving() && this.isValidJson()) {
          this.saveChanges();
        }
      }
    );
  }
}