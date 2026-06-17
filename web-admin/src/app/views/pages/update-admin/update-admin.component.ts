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
  SpinnerComponent
} from '@coreui/angular';
import { FormsModule, NgForm } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../../auth/auth.service';

@Component({
  selector: 'app-update-admin',
  templateUrl: './update-admin.component.html',
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
    SpinnerComponent,
    CommonModule
  ]
})
export class UpdateAdminComponent {
  authService = inject(AuthService);
  router = inject(Router);
  updateError: string | null = null;
  updateSuccess: boolean = false;
  isLoading = false;

  constructor() {}

  onUpdate(updateForm: NgForm): void {
    if (updateForm.invalid) {
      this.updateError = 'Please fill out all fields correctly.';
      return;
    }
    const { username, password, repeatPassword } = updateForm.value;

    if (!username && !password) {
      this.updateError = 'Please provide either a new username or password.';
      return;
    }

    if (password || repeatPassword) {
      if (password !== repeatPassword) {
        this.updateError = 'Passwords do not match.';
        return;
      }
    }

    this.isLoading = true;
    this.updateError = null;
    this.updateSuccess = false;

    this.authService.updateRoot({ username, password }).subscribe({
      next: (response) => {
        this.isLoading = false;
        this.updateSuccess = true;
        updateForm.resetForm();
      },
      error: (err) => {
        this.isLoading = false;
        this.updateError = err?.error?.error || 'Failed to update root user. Please try again.';
        console.error('Update root error:', err);
      }
    });
  }
}
