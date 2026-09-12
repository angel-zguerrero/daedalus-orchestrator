import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./workflows.component').then(m => m.WorkflowsComponent),
    data: {
      title: 'Workflows'
    }
  }
];
