import { Component, input, output, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';

@Component({
  selector: 'app-mongo-document-card',
  standalone: true,
  imports: [CommonModule, FormsModule, MonacoEditorModule],
  templateUrl: './mongo-document-card.component.html',
  styleUrl: './mongo-document-card.component.scss'
})
export class MongoDocumentCardComponent {
  document = input.required<Record<string, any>>();
  isSaving = input<boolean>(false);

  save = output<Record<string, any>>();
  reset = output<void>();

  editedJson = '';
  originalJson = '';
  currentOriginalJson = '';
  isExpanded = signal(false);
  documentKeys = signal<string[]>([]);

  constructor() {
    effect(() => {
      const doc = this.document();
      const sortedDoc = this.sortObjectKeys(doc);
      this.documentKeys.set(Object.keys(sortedDoc));
      this.originalJson = JSON.stringify(sortedDoc, null, 2);
      this.updateDisplayJson();
    });
  }

  private sortObjectKeys(obj: any): any {
    if (obj === null || typeof obj !== 'object' || Array.isArray(obj)) {
      return obj;
    }
    const keys = Object.keys(obj).sort();
    const sortedObj: any = {};

    // Ensure _id is first
    if (obj.hasOwnProperty('_id')) {
      sortedObj['_id'] = obj['_id'];
    }

    for (const key of keys) {
      if (key !== '_id') {
        sortedObj[key] = obj[key];
      }
    }
    return sortedObj;
  }

  updateDisplayJson(): void {
    const keys = this.documentKeys();
    if (!this.isExpanded() && keys.length > 15) {
      const doc = JSON.parse(this.originalJson);
      const truncated: any = {};
      keys.slice(0, 15).forEach(k => truncated[k] = doc[k]);
      this.currentOriginalJson = JSON.stringify(truncated, null, 2);
      this.editedJson = this.currentOriginalJson;
      this.editorOptions = { ...this.editorOptions, readOnly: true };
    } else {
      this.currentOriginalJson = this.originalJson;
      this.editedJson = this.currentOriginalJson;
      this.editorOptions = { ...this.editorOptions, readOnly: false };
    }
  }

  toggleExpand(): void {
    this.isExpanded.set(!this.isExpanded());
    this.updateDisplayJson();
  }

  hasMoreFields(): boolean {
    return this.documentKeys().length > 15;
  }

  isTruncated(): boolean {
    return !this.isExpanded() && this.hasMoreFields();
  }

  editorHeight(): number {
    const lines = this.editedJson.split('\n').length;
    return Math.max(100, lines * 19 + 24);
  }

  docId(): string {
    return String(this.document()['_id'] ?? 'unknown');
  }

  isDirty(): boolean {
    if (!this.editedJson || !this.currentOriginalJson) return false;
    try {
      const current = JSON.parse(this.editedJson);
      const original = JSON.parse(this.currentOriginalJson);
      return JSON.stringify(current) !== JSON.stringify(original);
    } catch {
      return true;
    }
  }

  onSave(): void {
    try {
      const edited = JSON.parse(this.editedJson);
      this.save.emit(edited);
    } catch (e) {
      alert('Invalid JSON');
    }
  }

  onReset(): void {
    this.editedJson = this.currentOriginalJson;
    this.reset.emit();
  }

  editorOptions = {
    theme: 'vs-dark',
    language: 'json',
    minimap: { enabled: false },
    fontSize: 13,
    fontFamily: "'JetBrains Mono', monospace",
    scrollBeyondLastLine: false,
    automaticLayout: true,
    padding: { top: 12, bottom: 12 },
    tabSize: 2,
    renderLineHighlight: 'all' as const,
    scrollbar: {
      vertical: 'hidden' as const,
      horizontal: 'hidden' as const,
      verticalScrollbarSize: 0,
      horizontalScrollbarSize: 0,
      alwaysConsumeMouseWheel: false
    },
    fixedOverflowWidgets: true,
    readOnly: false
  };
}
