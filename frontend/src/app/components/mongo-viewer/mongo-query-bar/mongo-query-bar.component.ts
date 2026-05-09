import { Component, output, signal, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

export interface MongoQueryOptions {
  filter: string;
  projection: string;
  sort: string;
  limit: number;
}

@Component({
  selector: 'app-mongo-query-bar',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './mongo-query-bar.component.html',
  styleUrl: './mongo-query-bar.component.scss'
})
export class MongoQueryBarComponent {
  filter = signal<string>('{}');
  projection = signal<string>('');
  sort = signal<string>('');
  limit = signal<number>(20);
  isExpanded = signal<boolean>(false);
  columns = input<string[]>([]);
  focusedInput = signal<'filter' | 'projection' | 'sort' | null>(null);

  queryExecuted = output<MongoQueryOptions>();

  toggleExpand(): void {
    this.isExpanded.set(!this.isExpanded());
  }

  getSuggestions(): string[] {
    const active = this.focusedInput();
    if (!active) return [];
    
    let val = '';
    if (active === 'filter') val = this.filter();
    else if (active === 'projection') val = this.projection();
    else if (active === 'sort') val = this.sort();
    
    const match = val.match(/([a-zA-Z0-9_]+)$/);
    const term = match ? match[1].toLowerCase() : '';
    
    return this.columns()
      .filter(c => c.toLowerCase().includes(term) && c !== term)
      .slice(0, 8);
  }

  insertSuggestion(col: string): void {
    const active = this.focusedInput();
    if (!active) return;
    
    let val = '';
    if (active === 'filter') val = this.filter();
    else if (active === 'projection') val = this.projection();
    else if (active === 'sort') val = this.sort();
    
    const match = val.match(/([a-zA-Z0-9_]+)$/);
    let newVal = val;
    if (match) {
        newVal = val.substring(0, val.length - match[1].length) + col + ': ';
    } else {
        newVal = val + (val.endsWith('{') || val.endsWith(' ') || val === '' ? '' : ', ') + col + ': ';
    }
    
    if (active === 'filter') this.filter.set(newVal);
    else if (active === 'projection') this.projection.set(newVal);
    else if (active === 'sort') this.sort.set(newVal);
  }

  onFocusOut(): void {
    this.focusedInput.set(null);
  }

  parseCompassQuery(str: string): string {
    if (!str || str.trim() === '') return '{}';
    let cleanStr = str.trim();
    if (!cleanStr.startsWith('{')) {
      cleanStr = '{' + cleanStr + '}';
    }
    try {
      JSON.parse(cleanStr);
      return cleanStr;
    } catch {
      try {
        const parsed = new Function('ObjectId', 'ISODate', 'return ' + cleanStr)(
          function(id: string) { return { $oid: id }; },
          function(date: string) { return { $date: date }; }
        );
        return JSON.stringify(parsed);
      } catch {
        return cleanStr;
      }
    }
  }

  onExecute(): void {
    this.queryExecuted.emit({
      filter: this.parseCompassQuery(this.filter()),
      projection: this.parseCompassQuery(this.projection()),
      sort: this.parseCompassQuery(this.sort()),
      limit: this.limit()
    });
  }
}
