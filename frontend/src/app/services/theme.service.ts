import { Injectable, signal } from '@angular/core';

export type Theme = 'dark' | 'light';

const STORAGE_KEY = 'unibase_theme';

/** Monaco has its own theme system, so the editor colors mirror the CSS tokens here. */
const MONACO_THEMES = {
    dark: {
        base: 'vs-dark',
        inherit: true,
        rules: [
            { token: 'comment', foreground: '637184', fontStyle: 'italic' },
            { token: 'keyword', foreground: 'F2A93B' },
            { token: 'string', foreground: '8FD19E' },
            { token: 'number', foreground: 'E8A0BF' },
            { token: 'predefined', foreground: '7FB2FF' },
            { token: 'type', foreground: 'B5B8FA' },
        ],
        colors: {
            'editor.background': '#0E131A',
            'editor.foreground': '#E4E9EE',
            'editorLineNumber.foreground': '#4A5666',
            'editorLineNumber.activeForeground': '#9AA7B5',
            'editor.selectionBackground': '#34405299',
            'editor.lineHighlightBackground': '#141B24',
            'editorCursor.foreground': '#F2A93B',
            'editorGutter.background': '#0E131A',
        }
    },
    light: {
        base: 'vs',
        inherit: true,
        rules: [
            { token: 'comment', foreground: '8792A0', fontStyle: 'italic' },
            { token: 'keyword', foreground: 'A85B00' },
            { token: 'string', foreground: '2D7A3C' },
            { token: 'number', foreground: 'A8326B' },
            { token: 'predefined', foreground: '2557B5' },
            { token: 'type', foreground: '4B4FC4' },
        ],
        colors: {
            'editor.background': '#FFFFFF',
            'editor.foreground': '#17202B',
            'editorLineNumber.foreground': '#A3ADB9',
            'editorLineNumber.activeForeground': '#4F5D6E',
            'editor.selectionBackground': '#C9D3F2',
            'editor.lineHighlightBackground': '#F3F5F7',
            'editorCursor.foreground': '#B36600',
            'editorGutter.background': '#FFFFFF',
        }
    }
};

@Injectable({ providedIn: 'root' })
export class ThemeService {
    readonly theme = signal<Theme>(this.initialTheme());

    constructor() {
        this.apply();
    }

    toggle(): void {
        this.theme.set(this.theme() === 'dark' ? 'light' : 'dark');
        try { localStorage.setItem(STORAGE_KEY, this.theme()); } catch { }
        this.apply();
    }

    /** Monaco theme name for editor options. */
    monacoTheme(): string {
        return `unibase-${this.theme()}`;
    }

    /** Registers both editor themes and applies the current one. Call from an editor's onInit. */
    applyMonacoTheme(): void {
        const monaco = (window as any).monaco;
        if (!monaco) return;
        monaco.editor.defineTheme('unibase-dark', MONACO_THEMES.dark);
        monaco.editor.defineTheme('unibase-light', MONACO_THEMES.light);
        monaco.editor.setTheme(this.monacoTheme());
    }

    private apply(): void {
        document.documentElement.setAttribute('data-theme', this.theme());
        this.applyMonacoTheme();
    }

    private initialTheme(): Theme {
        try {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored === 'dark' || stored === 'light') return stored;
        } catch { }
        return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
    }
}
