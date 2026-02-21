import { Routes } from '@angular/router';

export const routes: Routes = [
    {
        path: '',
        loadComponent: () => import('./job-workers.component').then(m => m.JobWorkersComponent),
        data: {
            title: 'Job Workers'
        }
    }
];
