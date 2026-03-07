import { ApplicationConfig, provideAppInitializer, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideMonacoEditor } from 'ngx-monaco-editor-v2';
import { inject } from '@angular/core';

import { routes } from './app.routes';
import { ProviderService } from './services/provider.service';
import { firstValueFrom } from 'rxjs';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideRouter(routes),
    provideHttpClient(),
    provideMonacoEditor({
      // Must be absolute so Web Workers can resolve the URL correctly.
      // Relative paths fail inside a WorkerGlobalScope because workers
      // resolve URLs relative to their own script, not the document root.
      baseUrl: `${window.location.origin}/assets/monaco/vs`,
      defaultOptions: {
        theme: 'vs-dark',
        automaticLayout: true
      }
    }),
    provideAppInitializer(() => {
      const providerService = inject(ProviderService);
      return firstValueFrom(providerService.loadProviders())
        .catch(error => {
          console.error('Failed to load providers:', error);
          return Promise.resolve();
        });
    })
  ]
};