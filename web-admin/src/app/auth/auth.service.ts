import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, BehaviorSubject, tap, catchError, of } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private apiUrl = '/admin-api/login'; // Corrected API endpoint
  private tokenKey = 'authToken';
  private isAuthenticatedSubject = new BehaviorSubject<boolean>(this.hasToken());

  public isAuthenticated$: Observable<boolean> = this.isAuthenticatedSubject.asObservable();

  constructor(private http: HttpClient, private router: Router) {}

  private hasToken(): boolean {
    return !!localStorage.getItem(this.tokenKey);
  }

  login(credentials: { username?: string, password?: string }): Observable<any> {
    const payload = {
      UsernameOrEmail: credentials.username,
      password: credentials.password
    };
    return this.http.post<any>(this.apiUrl, payload).pipe(
      tap(response => {
        // Assuming the token is in response.token or response.access_token
        const token = response.token || response.access_token;
        if (token) {
          localStorage.setItem(this.tokenKey, token);
          this.isAuthenticatedSubject.next(true);
          this.router.navigate(['/dashboard']);
        } else {
          // Handle cases where login is successful but no token is returned (should not happen ideally)
          console.error('Login successful but no token received.');
          this.isAuthenticatedSubject.next(false);
        }
      }),
      catchError(error => {
        console.error('Login failed:', error);
        this.isAuthenticatedSubject.next(false);
        // Rethrow the error or return a user-friendly error message
        return of(null); // Or throwError(() => new Error('Login failed'));
      })
    );
  }

  logout(): void {
    const logoutUrl = '/admin-api/logout'; // Define the logout endpoint
    // Make an HTTP POST request to the logout endpoint.
    // The request body might be empty or might require the token, depending on the API design.
    // For this example, we'll assume an empty body is sufficient.
    this.http.post(logoutUrl, {}).pipe(
      tap(() => {
        // This block will be executed on a successful logout response from the server.
        this.clearSession();
      }),
      catchError(error => {
        // This block will be executed if the API call fails.
        console.error('Logout API call failed:', error);
        // Even if the API call fails, we should still clear the local session to ensure the user is logged out client-side.
        this.clearSession();
        // Optionally, re-throw the error or return a user-friendly error message.
        return of(null); // Continue the observable chain.
      })
    ).subscribe();
  }

  private clearSession(): void {
    localStorage.removeItem(this.tokenKey);
    this.isAuthenticatedSubject.next(false);
    this.router.navigate(['/login']);
  }

  getToken(): string | null {
    return localStorage.getItem(this.tokenKey);
  }

  isLoggedIn(): boolean {
    const hasToken = this.hasToken();
    if (this.isAuthenticatedSubject.value !== hasToken) {
      this.isAuthenticatedSubject.next(hasToken);
    }
    return hasToken;
  }
}
