import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { UsersService, User } from './services/users.service';
import {
  TableModule,
  UtilitiesModule,
  ButtonModule,
  ModalModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
} from '@coreui/angular';
import {
  ReactiveFormsModule,
  FormsModule,
  FormBuilder,
  FormGroup,
  Validators,
  AbstractControl,
  ValidationErrors,
  ValidatorFn,
} from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { ErrorUtil } from '../../shared/utils/error.util';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../auth/auth.service';

export const passwordMatchValidator: ValidatorFn = (
  control: AbstractControl,
): ValidationErrors | null => {
  const password = control.get('Password');
  const repeatPassword = control.get('RepeatPassword');

  if (password && repeatPassword && password.value !== repeatPassword.value) {
    // We can also set error on the specific control
    repeatPassword.setErrors({ passwordMismatch: true });
    return { passwordMismatch: true };
  } else if (repeatPassword && repeatPassword.hasError('passwordMismatch')) {
    // Clear error if they match now
    const errors = { ...repeatPassword.errors };
    delete errors['passwordMismatch'];
    repeatPassword.setErrors(Object.keys(errors).length ? errors : null);
  }

  return null;
};

@Component({
  selector: 'app-users',
  templateUrl: './users.component.html',
  styleUrls: ['./users.component.scss'],
  standalone: true,
  imports: [
    AlertComponent,
    CommonModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    ReactiveFormsModule,
    FormsModule,
    SpinnerComponent,
    BadgeComponent,
    IconDirective,
  ],
})
export class UsersComponent implements OnInit {
  users: User[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;
  searchQuery = '';
  checkResolveDemo = false;
  isDemoMode = false;

  public createModalVisible = false;
  public editModalVisible = false;
  public deleteModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  userForm: FormGroup;
  userFormUpdate: FormGroup;
  selectedUser: any;

  constructor(
    private usersService: UsersService,
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router,
    public authService: AuthService
  ) {
    this.userForm = this.fb.group(
      {
        Username: ['', Validators.required],
        Email: ['', [Validators.required, Validators.email]],
        Password: ['', [Validators.required, Validators.minLength(6)]],
        RepeatPassword: ['', Validators.required],
      },
      { validators: passwordMatchValidator },
    );

    this.userFormUpdate = this.fb.group(
      {
        Username: ['', Validators.required],
        Email: ['', [Validators.required, Validators.email]],
        Password: ['', [Validators.minLength(6)]],
        RepeatPassword: [''],
      },
      { validators: passwordMatchValidator },
    );
  }

  ngOnInit(): void {
    this.cursors.push('');
    
    this.authService.isDemo$.subscribe(isDemo => {
      this.isDemoMode = isDemo;
    });

    this.loadUsers();

    this.route.queryParams.subscribe(params => {
      if (params['resolveDemo']) {
        if (this.users.length > 0) {
          const rootUser = this.users.find(u => u.IsRootUser);
          if (rootUser) {
            this.openEditModal(rootUser);
          }
        } else {
          this.checkResolveDemo = true;
        }

        this.router.navigate([], {
          relativeTo: this.route,
          queryParams: { resolveDemo: null },
          queryParamsHandling: 'merge',
          replaceUrl: true
        });
      }
    });
  }

  loadUsers(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    this.loading = true;
    this.usersService
      .getUsers(this.pageSize, cursor, this.searchQuery)
      .subscribe({
        next: (response) => {
          this.users = response.result.Entities;
          this.cursor = response.result.Cursor;
          this.loading = false;

          if (this.checkResolveDemo) {
            this.checkResolveDemo = false;
            const rootUser = this.users.find(u => u.IsRootUser);
            if (rootUser) {
              this.openEditModal(rootUser);
            }
          }
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
          this.loading = false;
        },
      });
  }

  searchUsers(): void {
    this.cursors = [''];
    this.loadUsers();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadUsers(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.loadUsers(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.userForm.reset();
  }

  openEditModal(user: User): void {
    this.selectedUser = user;
    this.userFormUpdate.reset();
    this.userFormUpdate.patchValue({
      Username: user.Username,
      Email: user.Email,
      Password: '',
      RepeatPassword: '',
    });
    this.editModalVisible = true;
  }

  openDeleteModal(user: User): void {
    if (user.IsRootUser) {
      return; // Safety guard
    }
    this.selectedUser = user;
    this.deleteModalVisible = true;
  }

  createUser(): void {
    if (this.userForm.valid) {
      this.usersService.createUser(this.userForm.value).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadUsers();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        },
      });
    } else {
      this.userForm.markAllAsTouched();
    }
  }

  updateUser(): void {
    if (this.userFormUpdate.valid) {
      const userData = this.userFormUpdate.getRawValue();
      this.usersService
        .updateUser(this.selectedUser.ID, userData)
        .subscribe({
          next: () => {
            this.editModalVisible = false;
            this.loadUsers();
            this.showAlert = false;
            this.authService.refreshAuthStatus().subscribe();
          },
          error: (error) => {
            this.showAlert = true;
            this.errorMessage = ErrorUtil.formatErrorMessage(error);
          },
        });
    } else {
      this.userFormUpdate.markAllAsTouched();
    }
  }

  deleteUser(): void {
    this.usersService.deleteUser(this.selectedUser.ID).subscribe({
      next: () => {
        this.deleteModalVisible = false;
        this.loadUsers();
        this.showAlert = false;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      },
    });
  }
}
