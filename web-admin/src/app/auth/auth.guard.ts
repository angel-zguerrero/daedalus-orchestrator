import { Injectable } from '@angular/core';
import { CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot, UrlTree } from '@angular/router';
import { Observable } from 'rxjs';
import { map, take } from 'rxjs/operators';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class AuthGuard implements CanActivate {

  constructor(private authService: AuthService, private router: Router) {}

  canActivate(
    next: ActivatedRouteSnapshot,
    state: RouterStateSnapshot): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {

    return this.authService.isAuthenticated$.pipe(
      take(1), // Take the latest value and complete
      map(isAuthenticated => {
        if (isAuthenticated) {
          return true;
        }
        // Not authenticated, redirect to login page
        return this.router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
      })
    );
  }
}

// Functional guard alternative (often preferred in modern Angular with standalone components)
import { inject } from '@angular/core';
import { CanActivateFn } from '@angular/router';

export const authGuardFn: CanActivateFn = (
  route: ActivatedRouteSnapshot,
  state: RouterStateSnapshot
): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree => {
  const authService = inject(AuthService);
  const router = inject(Router);

  return authService.isLoggedIn() // Using isLoggedIn for a direct boolean check, or use isAuthenticated$ for reactive check
    ? true
    : router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
};

export const setupGuardFn: CanActivateFn = (
  route: ActivatedRouteSnapshot,
  state: RouterStateSnapshot
): Observable<boolean | UrlTree> => {
  const authService = inject(AuthService);
  const router = inject(Router);

  return authService.checkRootExists().pipe(
    map(hasRoot => {
      const isSetupRoute = state.url.startsWith('/setup');
      if (!hasRoot) {
        if (!isSetupRoute) {
          return router.createUrlTree(['/setup']);
        }
        return true;
      } else {
        if (isSetupRoute) {
          return router.createUrlTree(['/login']);
        }
        return true;
      }
    })
  );
};
