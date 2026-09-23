import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./activity-templates.component').then((m) => m.ActivityTemplatesComponent),
    data: {
      title: 'Activity Templates'
    }
  }
];
