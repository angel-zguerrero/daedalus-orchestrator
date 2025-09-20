import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { IconDirective } from '@coreui/icons-angular';
import {
  ButtonDirective,
  CardBodyComponent,
  CardComponent,
  CardGroupComponent,
  ColComponent,
  ContainerComponent,
  FormControlDirective,
  FormDirective,
  InputGroupComponent,
  InputGroupTextDirective,
  RowComponent,
  TextColorDirective,
  AlertComponent,
} from '@coreui/angular';
import { FormsModule, NgForm } from '@angular/forms'; // Import FormsModule
import { Router } from '@angular/router';
import { AuthService } from '../../../auth/auth.service'; // Import AuthService

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  standalone: true, // Mark as standalone
  imports: [
    ContainerComponent,
    RowComponent,
    ColComponent,
    CardGroupComponent,
    CardComponent,
    CardBodyComponent,
    FormDirective,
    InputGroupComponent,
    InputGroupTextDirective,
    IconDirective,
    FormControlDirective,
    ButtonDirective,
    FormsModule, // Add FormsModule here
    TextColorDirective,
    AlertComponent,
    CommonModule
  ]
})
export class LoginComponent {
  authService = inject(AuthService);
  router = inject(Router);
  loginError: string | null = null;
  isLoading = false;

  constructor() {}

  onLogin(loginForm: NgForm): void {
    if (loginForm.invalid) {
      this.loginError = 'Please enter username and password.';
      return;
    }
    this.isLoading = true;
    this.loginError = null;
    const { username, password } = loginForm.value;

    this.authService.login({ username, password }).subscribe({
      next: (response) => {
        this.isLoading = false;
        if (response && this.authService.isLoggedIn()) { // Check if response is not null (meaning success from catchError)
          this.router.navigate(['/dashboard']);
        } else {
          // Error is handled by the service, but we might want a specific message here if login returns null from catchError
          this.loginError = 'Invalid username or password. Please try again.';
        }
      },
      error: (err) => {
        // This block might not be reached if catchError in service returns of(null)
        this.isLoading = false;
        this.loginError = 'An unexpected error occurred. Please try again.';
        console.error('Login component error:', err);
      }
    });
  }
}
