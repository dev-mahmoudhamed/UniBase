import { Component, effect, inject, input, model, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import { ConnectionService } from '../../services/connection.service';
import { ThemeService } from '../../services/theme.service';

@Component({
    selector: 'app-query-editor',
    standalone: true,
    imports: [FormsModule, MonacoEditorModule],
    templateUrl: './query-editor.component.html',
    styleUrls: ['./query-editor.component.scss']
})
export class QueryEditorComponent {
    private connectionService = inject(ConnectionService);
    private themeService = inject(ThemeService);
    language = input<string>('sql');
    executeQuery = output<{ query: string; queryId: string; sessionId: string }>();
    cancelQuery = output<void>();
    running = input(false);

    editorOptions = {
        theme: this.themeService.monacoTheme(),
        language: this.language(),
        minimap: { enabled: false, scale: 1, side: 'right' },
        fontSize: 13,
        fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace",
        fontLigatures: true,
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on' as 'on',
        renderLineHighlight: 'all' as 'all',
        tabSize: 2,
        cursorBlinking: 'smooth' as 'smooth',
        cursorSmoothCaretAnimation: 'on' as 'on',
        smoothScrolling: true,
        padding: { top: 16, bottom: 16 },
        wordWrap: 'on' as 'on',
        formatOnPaste: true,
        formatOnType: true,
        folding: true,
        links: true,
        roundedSelection: true,
        letterSpacing: 0.5,
        lineHeight: 20,
        stickyScroll: { enabled: true },
        scrollbar: {
            vertical: 'visible' as 'visible',
            horizontal: 'visible' as 'visible',
            useShadows: false,
            verticalHasArrows: false,
            horizontalHasArrows: false,
            verticalScrollbarSize: 10,
            horizontalScrollbarSize: 10
        }
    };

    constructor() {
        effect(() => {
            const lang = this.language();
            this.editorOptions = { ...this.editorOptions, language: lang };
        });
    }

    queryText = model('');
    activeConnection = this.connectionService.activeConnection;
    sessionId = this.connectionService.sessionId;
    isLoadingMetadata = this.connectionService.isLoadingMetadata;

    onEditorInit(editor: any): void {
        const monaco = (window as any).monaco;
        if (monaco) {
            this.editor = editor;
            this.themeService.applyMonacoTheme();

            // Utilizing the 'editor' parameter to add a keybinding
            editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
                this.onExecute();
            });
        }
    }

    /** Inline message instead of a browser alert. */
    notice = signal<string | null>(null);
    /** Set when a Prod run is waiting for the user to confirm. */
    confirmProd = signal(false);
    private editor: any;

    onExecute(confirmed = false): void {
        if (this.running()) return;
        const connection = this.activeConnection();
        const sessionId = this.connectionService.sessionId();
        this.notice.set(null);
        this.confirmProd.set(false);

        if (!connection || !sessionId) {
            this.notice.set('Open a connection in the sidebar first.');
            return;
        }

        // Run the selection if there is one, otherwise the whole editor.
        const model = this.editor?.getModel();
        const selected = model && this.editor.getSelection() ? model.getValueInRange(this.editor.getSelection()) : '';
        const query = selected.trim() ? selected : this.queryText();
        if (!query.trim()) {
            this.notice.set('Write a query to run.');
            return;
        }

        if (connection.environment === 'prod' && !confirmed && changesData(query)) {
            this.confirmProd.set(true);
            return;
        }

        // A fresh ID per run: the server only reuses it while this run is being polled.
        this.executeQuery.emit({ query, queryId: newRunId(), sessionId });
    }
}

/** True if the script contains a statement that changes data or schema (comments ignored). */
function changesData(query: string): boolean {
    const code = query.replace(/--.*$/gm, '').replace(/\/\*[\s\S]*?\*\//g, '');
    return /\b(insert|update|delete|merge|drop|truncate|alter|create|grant|revoke|exec|execute)\b/i.test(code);
}

function newRunId(): string {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
}
