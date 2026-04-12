import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./tenants.component').then(m => m.TenantsComponent),
    data: {
      title: 'Tenants'
    }
  },
  {
    path: ':code/management',
    loadComponent: () => import('./tenant-management/tenant-management.component').then(m => m.TenantManagementComponent),
    data: {
      title: 'Tenant Management'
    }
  },
  {
    path: ':tenantCode/queues/:queueCode/:vnamespace/messages',
    loadComponent: () => import('./tenant-management/queue-messages/queue-messages.component').then(m => m.QueueMessagesComponent),
    data: {
      title: 'Queue Messages'
    }
  }
];
