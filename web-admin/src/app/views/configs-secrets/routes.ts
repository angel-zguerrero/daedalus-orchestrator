import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./configs-secrets.component').then(m => m.ConfigsSecretsComponent),
    data: {
      title: 'Global Configs & Secrets'
    }
  }
];
