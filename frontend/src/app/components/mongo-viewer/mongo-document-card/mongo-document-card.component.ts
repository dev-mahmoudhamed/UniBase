import { Component, computed, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import { ThemeService } from '../../../services/theme.service';

const PREVIEW_FIELDS = 12;

/**
 * One document. It shows a light field list with BSON types; the Monaco JSON
 * editor is mounted only while the document is being edited.
 */
@Component({
  selector: 'app-mongo-document-card',
  standalone: true,
  imports: [FormsModule, MonacoEditorModule],
  templateUrl: './mongo-document-card.component.html',
  styleUrl: './mongo-document-card.component.scss'
})
export class MongoDocumentCardComponent {
  private themeService = inject(ThemeService);

  document = input.required<Record<string, any>>();
  isSaving = input<boolean>(false);
  save = output<Record<string, any>>();

  editing = signal(false);
  showAll = signal(false);
  error = signal<string | null>(null);
  editedJson = '';
  private originalJson = '';

  private keys = computed(() =>
    Object.keys(this.document()).sort((a, b) => a === '_id' ? -1 : b === '_id' ? 1 : a.localeCompare(b)));

  fields = computed(() => {
    const doc = this.document();
    const keys = this.showAll() ? this.keys() : this.keys().slice(0, PREVIEW_FIELDS);
    return keys.map(key => ({ key, value: preview(doc[key]), type: bsonType(doc[key]) }));
  });
  hiddenCount = computed(() => this.showAll() ? 0 : Math.max(0, this.keys().length - PREVIEW_FIELDS));

  editorOptions = {
    theme: this.themeService.monacoTheme(),
    language: 'json',
    minimap: { enabled: false },
    fontSize: 13,
    fontFamily: "'JetBrains Mono', monospace",
    scrollBeyondLastLine: false,
    automaticLayout: true,
    padding: { top: 12, bottom: 12 },
    tabSize: 2,
    scrollbar: { alwaysConsumeMouseWheel: false },
    fixedOverflowWidgets: true
  };

  constructor() {
    // A new or saved document always opens in the read view.
    effect(() => {
      this.document();
      this.editing.set(false);
    });
  }

  docId(): string {
    const id = this.document()['_id'];
    if (id === undefined || id === null) return 'no _id';
    if (typeof id === 'object' && typeof id.$oid === 'string') return id.$oid;
    return typeof id === 'string' ? id : JSON.stringify(id);
  }

  edit(): void {
    const doc = this.document();
    const ordered = Object.fromEntries(this.keys().map(k => [k, doc[k]]));
    this.originalJson = JSON.stringify(ordered, null, 2);
    this.editedJson = this.originalJson;
    this.error.set(null);
    this.editing.set(true);
  }

  isDirty(): boolean {
    return this.editedJson !== this.originalJson;
  }

  editorHeight(): number {
    return Math.min(560, Math.max(120, this.editedJson.split('\n').length * 19 + 24));
  }

  onSave(): void {
    try {
      this.save.emit(JSON.parse(this.editedJson));
      this.error.set(null);
    } catch (e) {
      this.error.set(`This is not valid JSON: ${(e as Error).message}`);
    }
  }
}

/** BSON type name from the relaxed Extended JSON the server sends. */
function bsonType(v: unknown): string {
  if (v === null) return 'Null';
  if (Array.isArray(v)) return `Array(${v.length})`;
  if (typeof v === 'object') {
    const o = v as Record<string, unknown>;
    if ('$oid' in o) return 'ObjectId';
    if ('$date' in o) return 'Date';
    if ('$numberDecimal' in o) return 'Decimal128';
    if ('$numberLong' in o) return 'Int64';
    if ('$binary' in o) return 'Binary';
    if ('$timestamp' in o) return 'Timestamp';
    return 'Object';
  }
  if (typeof v === 'number') return 'Number';
  if (typeof v === 'string') return 'String';
  if (typeof v === 'boolean') return 'Boolean';
  return typeof v;
}

/** One-line value for the read view. */
function preview(v: unknown): string {
  if (v && typeof v === 'object' && !Array.isArray(v)) {
    const o = v as Record<string, any>;
    if (typeof o['$oid'] === 'string') return o['$oid'];
    if ('$date' in o) return typeof o['$date'] === 'string' ? o['$date'] : JSON.stringify(o['$date']);
    if (typeof o['$numberDecimal'] === 'string') return o['$numberDecimal'];
    if (typeof o['$numberLong'] === 'string') return o['$numberLong'];
  }
  return typeof v === 'string' ? `"${v}"` : JSON.stringify(v);
}
