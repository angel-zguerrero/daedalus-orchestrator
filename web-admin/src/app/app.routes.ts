import { Routes } from '@angular/router';
import { authGuardFn } from './auth/auth.guard'; // Import the functional auth guard

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
    canActivate: [authGuardFn], // Apply AuthGuard here
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
      }
    ]
  },
  {
    path: '404',
    loadComponent: () => import('./views/pages/page404/page404.component').then(m => m.Page404Component),
    data: {
      title: 'Page 404'
    }
  },
  {
    path: '500',
    loadComponent: () => import('./views/pages/page500/page500.component').then(m => m.Page500Component),
    data: {
      title: 'Page 500'
    }
  },
  {
    path: 'login',
    loadComponent: () => import('./views/pages/login/login.component').then(m => m.LoginComponent),
    data: {
      title: 'Login Page'
    }
  },
  {
    path: 'register',
    loadComponent: () => import('./views/pages/register/register.component').then(m => m.RegisterComponent),
    data: {
      title: 'Register Page'
    }
  },
  // The 'pages' route that was previously a child of DefaultLayoutComponent seems to define
  // login, register, 404, 500. These are typically standalone.
  // If there's a separate '/pages' path that needs to exist and be protected, it should be defined within the guarded DefaultLayoutComponent.
  // For now, I am assuming the individual page routes (login, register, 404, 500) are sufficient as top-level.
  {
    path: 'pages', // This path seems to be for non-auth pages based on its typical content
    loadChildren: () => import('./views/pages/routes').then((m) => m.routes)
  },
  { path: '**', redirectTo: 'dashboard' } // If not logged in, guard on 'dashboard' (via DefaultLayout) will redirect to login.
];
