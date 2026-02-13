import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    data: {
      title: 'Cluster'
    },
    children: [
      {
        path: '',
        redirectTo: 'management',
        pathMatch: 'full'
      },
      {
        path: 'management',
        loadComponent: () => import('./cluster.component').then(m => m.ClusterComponent),
        data: {
          title: 'Cluster Management'
        }
      }
    ]
  }
];