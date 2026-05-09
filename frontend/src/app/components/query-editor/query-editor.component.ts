import { Component, effect, inject, input, model, output } from '@angular/core';
import { hashQueryId } from '../../utils/hash.utils';

import { FormsModule } from '@angular/forms';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import { ConnectionService } from '../../services/connection.service';

@Component({
    selector: 'app-query-editor',
    standalone: true,
    imports: [FormsModule, MonacoEditorModule],
    templateUrl: './query-editor.component.html',
    styleUrls: ['./query-editor.component.scss']
})
export class QueryEditorComponent {
    private connectionService = inject(ConnectionService);
    language = input<string>('sql');
    executeQuery = output<{ query: string; queryId: string; sessionId: string }>();

    editorOptions = {
        theme: 'unibase-dark',
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
            monaco.editor.defineTheme('unibase-dark', {
                base: 'vs-dark',
                inherit: true,
                rules: [
                    { token: 'comment', foreground: '5c6370', fontStyle: 'italic' },
                    { token: 'keyword', foreground: 'c678dd', fontStyle: 'bold' },
                    { token: 'string', foreground: '98c379' },
                    { token: 'number', foreground: 'd19a66' },
                    { token: 'operator', foreground: '56b6c2' },
                    { token: 'identifier', foreground: '61afef' },
                    { token: 'type', foreground: 'e5c07b' },
                    { token: 'function', foreground: '61afef', fontStyle: 'bold' },
                    { token: 'variable', foreground: 'e06c75' },
                    { token: 'constant', foreground: 'd19a66' },
                    { token: 'string.escape', foreground: '56b6c2' },
                ],
                colors: {
                    'editor.background': '#0d0d0f',           // Near-black background
                    'editor.foreground': '#abb2bf',           // One Dark text
                    'editorLineNumber.foreground': '#4b4b4f', // Muted line numbers
                    'editorLineNumber.activeForeground': '#abb2bf',
                    'editor.selectionBackground': '#3e445166', // Subtle selection
                    'editor.lineHighlightBackground': '#161618', // Very subtle line highlight
                    'editorCursor.foreground': '#a78bfa',     // Violet cursor
                    'editor.findMatchBackground': '#42557b',
                    'editorBracketMatch.background': '#515a6b',
                    'editorBracketMatch.border': '#888888',
                    'editorGutter.background': '#0d0d0f',
                }
            });
            monaco.editor.setTheme('unibase-dark');

            // Utilizing the 'editor' parameter to add a keybinding
            editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
                this.onExecute();
            });
        }
    }

    onExecute(): void {
        const connection = this.activeConnection();
        const currentQuery = this.queryText();
        const sessionId = this.connectionService.sessionId();

        if (!connection) {
            alert('Please select a database connection first');
            return;
        }

        if (!sessionId) {
            alert('Connection session not initialized. Please reconnect.');
            return;
        }

        if (!currentQuery.trim()) {
            alert('Please enter a query');
            return;
        }
        const queryId = hashQueryId(currentQuery);
        this.executeQuery.emit({ query: currentQuery, queryId, sessionId });
    }
}
