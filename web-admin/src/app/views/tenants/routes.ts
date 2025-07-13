import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./tenants.component').then(m => m.TenantsComponent),
    data: {
      title: 'Tenants'
    }
  }
];
