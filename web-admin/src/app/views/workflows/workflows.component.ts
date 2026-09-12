import { Component, OnInit, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
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
  NavModule,
  TabsModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition } from './services/workflows.service';
import { ErrorUtil } from '../../shared/utils/error.util';

@Component({
  selector: 'app-workflows',
  templateUrl: './workflows.component.html',
  styleUrls: ['./workflows.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
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
    NavModule,
    TabsModule,
    IconDirective
  ]
})
export class WorkflowsComponent implements OnInit, OnChanges {
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';

  workflows: WorkflowDefinition[] = [];
  filteredWorkflows: WorkflowDefinition[] = [];
  loading: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  searchQuery: string = '';
  selectedStatusFilter: 'all' | 'active' | 'inactive' = 'all';

  // Pagination
  cursor: string = '';
  nextCursor: string = '';
  cursors: string[] = [''];
  pageSize: number = 20;

  // Create / Edit Modal
  showModal: boolean = false;
  workflowForm: FormGroup;
  isEditing: boolean = false;
  editingWorkflowId: string = '';

  // Detail / Payload Modal
  showDetailModal: boolean = false;
  selectedWorkflow: WorkflowDefinition | null = null;
  payloadDisplayText: string = '';

  // Delete Confirm Modal
  showDeleteModal: boolean = false;
  workflowToDelete: WorkflowDefinition | null = null;

  constructor(
    private fb: FormBuilder,
    private workflowsService: WorkflowsService
  ) {
    this.workflowForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      description: [''],
      version: [1, [Validators.required, Validators.min(1)]],
      payloadFormat: ['json', Validators.required],
      payload: ['{}'],
      maxDurationSeconds: [3600, [Validators.required, Validators.min(0)]],
      isActive: [true]
    });
  }

  ngOnInit(): void {
    this.loadWorkflows();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['tenantCode'] || changes['scope']) {
      this.cursor = '';
      this.cursors = [''];
      this.loadWorkflows();
    }
  }

  private decodePayload(payloadRaw: any): string {
    if (!payloadRaw) return '';
    if (typeof payloadRaw === 'string') {
      try {
        return atob(payloadRaw);
      } catch {
        return payloadRaw;
      }
    }
    return JSON.stringify(payloadRaw, null, 2);
  }

  private normalizeWorkflow(w: any): WorkflowDefinition {
    return {
      id: w.id || w.ID || '',
      code: w.code || w.Code || '',
      vnamespace: w.vnamespace || w.VNamespace || '',
      name: w.name || w.Name || w.code || w.Code || '',
      description: w.description || w.Description || '',
      version: w.version || w.Version || 1,
      payload: w.payload || w.Payload || '',
      payloadFormat: (w.payloadFormat || w.PayloadFormat || 'json').toLowerCase() as 'json' | 'yaml',
      maxDurationSeconds: w.maxDurationSeconds || w.MaxDurationSeconds || 0,
      isActive: w.isActive !== undefined ? w.isActive : (w.IsActive !== undefined ? w.IsActive : true),
      scope: (w.scope || w.Scope || this.scope).toLowerCase() as 'global' | 'tenant',
      tenantId: w.tenantId || w.TenantID || '',
      createdAt: w.createdAt || w.CreatedAt || '',
      updatedAt: w.updatedAt || w.UpdatedAt || ''
    };
  }

  loadWorkflows(): void {
    this.loading = true;
    this.showAlert = false;
    this.errorMessage = '';

    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflows(this.pageSize, this.cursor)
      : this.workflowsService.getTenantWorkflows(this.tenantCode, this.pageSize, this.cursor);

    req$.subscribe({
      next: (res) => {
        const rawEntities = res.Entities || res.entities || [];
        this.workflows = rawEntities.map((w: any) => this.normalizeWorkflow(w));
        this.nextCursor = res.Cursor || res.cursor || '';
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.loading = false;
      }
    });
  }

  applyFilters(): void {
    let result = [...this.workflows];

    if (this.selectedStatusFilter === 'active') {
      result = result.filter(w => w.isActive);
    } else if (this.selectedStatusFilter === 'inactive') {
      result = result.filter(w => !w.isActive);
    }

    if (this.searchQuery.trim()) {
      const q = this.searchQuery.toLowerCase().trim();
      result = result.filter(w =>
        w.code.toLowerCase().includes(q) ||
        (w.name && w.name.toLowerCase().includes(q)) ||
        (w.description && w.description.toLowerCase().includes(q))
      );
    }

    this.filteredWorkflows = result;
  }

  searchWorkflows(): void {
    this.applyFilters();
  }

  nextPage(): void {
    if (this.nextCursor) {
      this.cursors.push(this.nextCursor);
      this.cursor = this.nextCursor;
      this.loadWorkflows();
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.cursor = this.cursors[this.cursors.length - 1];
      this.loadWorkflows();
    }
  }

  openCreateModal(): void {
    this.isEditing = false;
    this.editingWorkflowId = '';
    this.workflowForm.reset({
      version: 1,
      payloadFormat: 'json',
      payload: '{\n  "steps": []\n}',
      maxDurationSeconds: 3600,
      isActive: true
    });
    this.workflowForm.get('code')?.enable();
    this.showModal = true;
  }

  openEditModal(wf: WorkflowDefinition): void {
    this.isEditing = true;
    this.editingWorkflowId = wf.id || '';
    const decodedPayload = this.decodePayload(wf.payload);
    this.workflowForm.patchValue({
      code: wf.code,
      name: wf.name,
      description: wf.description,
      version: wf.version,
      payloadFormat: wf.payloadFormat,
      payload: decodedPayload,
      maxDurationSeconds: wf.maxDurationSeconds,
      isActive: wf.isActive
    });
    this.workflowForm.get('code')?.disable();
    this.showModal = true;
  }

  openDetailModal(wf: WorkflowDefinition): void {
    this.selectedWorkflow = wf;
    this.payloadDisplayText = this.decodePayload(wf.payload);
    this.showDetailModal = true;
  }

  saveWorkflow(): void {
    if (this.workflowForm.invalid) return;

    const val = this.workflowForm.getRawValue();
    const payload: Partial<WorkflowDefinition> = {
      code: val.code ? val.code.trim() : '',
      name: val.name ? val.name.trim() : '',
      description: val.description,
      version: Number(val.version),
      payloadFormat: val.payloadFormat,
      payload: val.payload,
      maxDurationSeconds: Number(val.maxDurationSeconds),
      isActive: Boolean(val.isActive),
      scope: this.scope
    };

    if (this.isEditing) {
      const update$ = this.scope === 'global'
        ? this.workflowsService.updateGlobalWorkflow(this.editingWorkflowId, payload)
        : this.workflowsService.updateTenantWorkflow(this.tenantCode, this.editingWorkflowId, payload);

      update$.subscribe({
        next: () => {
          this.showModal = false;
          this.successMsg = 'Workflow definition updated successfully.';
          this.loadWorkflows();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    } else {
      const create$ = this.scope === 'global'
        ? this.workflowsService.createGlobalWorkflow(payload)
        : this.workflowsService.createTenantWorkflow(this.tenantCode, payload);

      create$.subscribe({
        next: () => {
          this.showModal = false;
          this.successMsg = 'Workflow definition created successfully.';
          this.loadWorkflows();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    }
  }

  confirmDelete(wf: WorkflowDefinition): void {
    this.workflowToDelete = wf;
    this.showDeleteModal = true;
  }

  deleteWorkflow(): void {
    if (!this.workflowToDelete || !this.workflowToDelete.id) return;

    const del$ = this.scope === 'global'
      ? this.workflowsService.deleteGlobalWorkflow(this.workflowToDelete.id)
      : this.workflowsService.deleteTenantWorkflow(this.tenantCode, this.workflowToDelete.id);

    del$.subscribe({
      next: () => {
        this.showDeleteModal = false;
        this.workflowToDelete = null;
        this.successMsg = 'Workflow definition deleted successfully.';
        this.loadWorkflows();
        setTimeout(() => this.successMsg = '', 3000);
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
      }
    });
  }
}
