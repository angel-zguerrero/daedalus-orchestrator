import { Component, OnInit, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  TableModule,
  ButtonModule,
  ModalModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  TooltipModule,
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { OAuthAppsService, OAuthApp, OAuthAppCreated } from './services/oauth-apps.service';

interface ScopeGroup {
  resource: string;
  actions: string[];
}

@Component({
  selector: 'app-service-accounts',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TableModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    TooltipModule,
    IconDirective,
  ],
  templateUrl: './service-accounts.component.html',
  styleUrls: ['./service-accounts.component.scss'],
})
export class ServiceAccountsComponent implements OnInit {
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';

  apps: OAuthApp[] = [];
  loading: boolean = false;
  error: string = '';

  showCreateForm: boolean = false;
  submitting: boolean = false;
  newAppName: string = '';
  newAppDescription: string = '';
  selectedScopes: Set<string> = new Set();

  deletingId: string | null = null;
  copiedMessage: string = '';
  copiedField: string | null = null;

  secretModal = {
    visible: false,
    appName: '',
    clientId: '',
    clientSecret: '',
  };

  scopeGroups: ScopeGroup[] = [
    { resource: 'queues', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
    { resource: 'exchanges', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
    { resource: 'bindings', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
    { resource: 'workflows', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
    { resource: 'tenants', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
    { resource: 'users', actions: ['admin', 'create', 'delete', 'edit', 'list'] },
  ];

  get visibleScopeGroups(): ScopeGroup[] {
    if (this.scope === 'tenant') {
      return this.scopeGroups.filter(
        (g) => g.resource !== 'tenants' && g.resource !== 'users'
      );
    }
    return this.scopeGroups;
  }

  constructor(private oauthService: OAuthAppsService) {}

  ngOnInit(): void {
    if (this.scope === 'global' || this.tenantCode) {
      this.loadApps();
    }
  }

  loadApps(): void {
    this.loading = true;
    this.error = '';
    const req$ = this.scope === 'global'
      ? this.oauthService.listGlobal()
      : this.oauthService.list(this.tenantCode);

    req$.subscribe({
      next: (res) => {
        this.apps = res.items || [];
        this.loading = false;
      },
      error: (err) => {
        this.error = err?.error?.error || 'Failed to load Service Accounts';
        this.loading = false;
      },
    });
  }

  openCreateForm(): void {
    this.showCreateForm = true;
    this.newAppName = '';
    this.newAppDescription = '';
    this.selectedScopes.clear();
  }

  cancelCreate(): void {
    this.showCreateForm = false;
    this.newAppName = '';
    this.newAppDescription = '';
    this.selectedScopes.clear();
  }

  toggleScope(scope: string, isAdmin: boolean, resource: string): void {
    if (isAdmin) {
      const adminScope = `${resource}:admin`;
      if (this.selectedScopes.has(adminScope)) {
        this.selectedScopes.delete(adminScope);
      } else {
        for (const action of ['create', 'delete', 'edit', 'list']) {
          this.selectedScopes.delete(`${resource}:${action}`);
        }
        this.selectedScopes.add(adminScope);
      }
    } else {
      if (this.selectedScopes.has(scope)) {
        this.selectedScopes.delete(scope);
      } else {
        this.selectedScopes.add(scope);
      }
    }
  }

  isAdminSelected(resource: string): boolean {
    return this.selectedScopes.has(`${resource}:admin`);
  }

  submitCreate(): void {
    if (!this.newAppName.trim()) return;
    this.submitting = true;

    const payload = {
      name: this.newAppName.trim(),
      description: this.newAppDescription.trim(),
      allowedScopes: Array.from(this.selectedScopes),
    };

    const req$ = this.scope === 'global'
      ? this.oauthService.createGlobal(payload)
      : this.oauthService.create(this.tenantCode, payload);

    req$.subscribe({
      next: (res: OAuthAppCreated) => {
        this.submitting = false;
        this.showCreateForm = false;
        this.openSecretModal(res);
        this.loadApps();
      },
      error: (err) => {
        this.error = err?.error?.error || 'Failed to create Service Account';
        this.submitting = false;
      },
    });
  }

  rotateSecret(app: OAuthApp): void {
    const req$ = this.scope === 'global'
      ? this.oauthService.rotateSecretGlobal(app.id)
      : this.oauthService.rotateSecret(this.tenantCode, app.id);

    req$.subscribe({
      next: (res: OAuthAppCreated) => {
        this.openSecretModal(res);
      },
      error: (err) => {
        this.error = err?.error?.error || 'Failed to rotate secret';
      },
    });
  }

  confirmDelete(appId: string): void {
    this.deletingId = appId;
  }

  cancelDelete(): void {
    this.deletingId = null;
  }

  executeDelete(appId: string): void {
    const req$ = this.scope === 'global'
      ? this.oauthService.deleteGlobal(appId)
      : this.oauthService.delete(this.tenantCode, appId);

    req$.subscribe({
      next: () => {
        this.deletingId = null;
        this.loadApps();
      },
      error: (err) => {
        this.error = err?.error?.error || 'Failed to delete Service Account';
        this.deletingId = null;
      },
    });
  }

  openSecretModal(res: OAuthAppCreated): void {
    this.secretModal = {
      visible: true,
      appName: res.name,
      clientId: res.clientId,
      clientSecret: res.clientSecret,
    };
  }

  closeSecretModal(): void {
    this.secretModal = {
      visible: false,
      appName: '',
      clientId: '',
      clientSecret: '',
    };
    this.copiedMessage = '';
    this.copiedField = null;
  }

  copyToClipboard(text: string, field?: string): void {
    navigator.clipboard.writeText(text);
    this.copiedField = field || 'copied';
    this.copiedMessage = 'Copied to clipboard!';
    setTimeout(() => {
      this.copiedMessage = '';
      this.copiedField = null;
    }, 2000);
  }
}
