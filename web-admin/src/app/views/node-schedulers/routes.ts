import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    data: {
      title: 'Node Schedulers'
    },
    children: [
      {
        path: '',
        redirectTo: 'list',
        pathMatch: 'full'
      },
      {
        path: 'list',
        loadComponent: () => import('./node-schedulers.component').then(m => m.NodeSchedulersComponent),
        data: {
          title: 'Node Schedulers List'
        }
      }
    ]
  }
];
