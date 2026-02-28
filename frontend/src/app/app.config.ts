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
      baseUrl: 'assets/monaco/vs',
      defaultOptions: {
        theme: 'vs-dark',
        automaticLayout: true
      }
    }),
    // Initialize providers on app startup using new API
    provideAppInitializer(() => {
      const providerService = inject(ProviderService);
      return firstValueFrom(providerService.loadProviders())
        .catch(error => {
          console.error('Failed to load providers:', error);
          // Don't fail app startup, just log the error
          return Promise.resolve();
        });
    })
  ]
};