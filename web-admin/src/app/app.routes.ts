import { Routes } from '@angular/router';
import { authGuardFn, setupGuardFn } from './auth/auth.guard'; // Import the functional guards

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'dashboard', // This will be caught by the guard if not authenticated
    pathMatch: 'full',
    data: {
      title: 'Daedalus'
    }
  },
  {
    path: '', // This route group contains all authenticated pages
    loadComponent: () => import('./layout').then(m => m.DefaultLayoutComponent),
    canActivate: [setupGuardFn, authGuardFn], // Apply SetupGuard and AuthGuard here
    data: {
      title: 'Home'
    },
    children: [
      {
        path: 'dashboard',
        loadChildren: () => import('./views/dashboard/routes').then((m) => m.routes)
      },
      {
        path: 'tenants',
        loadChildren: () => import('./views/tenants/routes').then((m) => m.routes)
      },
      {
        path: 'cluster',
        loadChildren: () => import('./views/cluster/routes').then((m) => m.routes)
      },
      {
        path: 'node-schedulers',
        loadChildren: () => import('./views/node-schedulers/routes').then((m) => m.routes)
      },
      {
        path: 'job-workers',
        loadChildren: () => import('./views/job-workers/routes').then((m) => m.routes)
      }
    ]
  },
  {
    path: '404',
    loadComponent: () => import('./views/pages/page404/page404.component').then(m => m.Page404Component),
    canActivate: [setupGuardFn],
    data: {
      title: 'Page 404'
    }
  },
  {
    path: '500',
    loadComponent: () => import('./views/pages/page500/page500.component').then(m => m.Page500Component),
    canActivate: [setupGuardFn],
    data: {
      title: 'Page 500'
    }
  },
  {
    path: 'login',
    loadComponent: () => import('./views/pages/login/login.component').then(m => m.LoginComponent),
    canActivate: [setupGuardFn],
    data: {
      title: 'Login Page'
    }
  },
  {
    path: 'setup',
    loadComponent: () => import('./views/pages/setup/setup.component').then(m => m.SetupComponent),
    canActivate: [setupGuardFn],
    data: {
      title: 'Setup Root User'
    }
  },
  { path: '**', redirectTo: 'dashboard' } // If not logged in, guard on 'dashboard' (via DefaultLayout) will redirect to login.
];
