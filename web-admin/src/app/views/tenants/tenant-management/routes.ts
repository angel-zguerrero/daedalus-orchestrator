import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: ':id/management',
    loadComponent: () => import('./tenant-management.component').then(m => m.TenantManagementComponent),
    data: {
      title: 'Tenant Management'
    }
  }
];
