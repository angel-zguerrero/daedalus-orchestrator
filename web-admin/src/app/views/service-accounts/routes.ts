import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./service-accounts.component').then(m => m.ServiceAccountsComponent),
    data: {
      title: 'Global Service Accounts'
    }
  }
];
