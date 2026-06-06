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
import { FormsModule, NgForm } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../../auth/auth.service';

@Component({
  selector: 'app-setup',
  templateUrl: './setup.component.html',
  standalone: true,
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
    FormsModule,
    TextColorDirective,
    AlertComponent,
    CommonModule
  ]
})
export class SetupComponent {
  authService = inject(AuthService);
  router = inject(Router);
  setupError: string | null = null;
  isLoading = false;

  constructor() {}

  onSetup(setupForm: NgForm): void {
    if (setupForm.invalid) {
      this.setupError = 'Please fill out all fields correctly.';
      return;
    }
    const { username, password, repeatPassword } = setupForm.value;
    if (password !== repeatPassword) {
      this.setupError = 'Passwords do not match.';
      return;
    }

    this.isLoading = true;
    this.setupError = null;

    this.authService.setupRoot({ username, password }).subscribe({
      next: (response) => {
        this.isLoading = false;
        this.router.navigate(['/login']);
      },
      error: (err) => {
        this.isLoading = false;
        this.setupError = err?.error?.error || 'Failed to configure root user. Please try again.';
        console.error('Setup root error:', err);
      }
    });
  }
}
